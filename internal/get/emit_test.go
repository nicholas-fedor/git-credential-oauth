// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get

import (
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/nicholas-fedor/git-credential-oauth/internal/credential"
	"github.com/nicholas-fedor/git-credential-oauth/internal/forge"
	"github.com/nicholas-fedor/git-credential-oauth/internal/oauth"
)

// TestEmitExpiryClamp covers omission and clamping of password_expiry_utc.
func TestEmitExpiryClamp(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.September, 30, 12, 0, 0, 0, time.UTC)
	margin := DefaultExpiryMargin

	tests := []struct {
		name   string
		expiry time.Time
		want   string
		omit   bool
	}{
		{
			name:   "zero expiry omits the key",
			expiry: time.Time{},
			omit:   true,
		},
		{
			name:   "inside margin clamps to now",
			expiry: now.Add(30 * time.Second),
			want:   strconv.FormatInt(now.Unix(), 10),
		},
		{
			name:   "already past clamps to now",
			expiry: now.Add(-time.Hour),
			want:   strconv.FormatInt(now.Unix(), 10),
		},
		{
			name:   "outside margin subtracts the margin",
			expiry: now.Add(time.Hour),
			want:   strconv.FormatInt(now.Add(time.Hour).Add(-margin).Unix(), 10),
		},
		{
			name:   "exact boundary is not clamped further",
			expiry: now.Add(margin),
			want:   strconv.FormatInt(now.Unix(), 10),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := emit(
				baseRequest(),
				detectedFor(forge.KindGitHub, "github.com"),
				tokenAt(tt.expiry),
				Options{},
				now,
				margin,
			)
			expiry, found := pairValue(got, credential.PasswordExpiryUTC)

			if tt.omit {
				if found {
					t.Fatalf("password_expiry_utc = %q, want omitted", expiry)
				}

				return
			}

			if !found {
				t.Fatal("password_expiry_utc omitted")
			}

			if expiry != tt.want {
				t.Fatalf("password_expiry_utc = %s, want %s", expiry, tt.want)
			}
		})
	}
}

// TestEmitAuthtypeAndBearer covers capability gating and bearer versus password.
func TestEmitAuthtypeAndBearer(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.September, 30, 12, 0, 0, 0, time.UTC)
	token := tokenAt(time.Time{})
	token.AccessToken = "access-token"
	token.RefreshToken = "refresh-token"

	tests := []struct {
		name       string
		host       string
		username   string
		capability []string
		want       []credential.Pair
		kind       forge.Kind
		useBearer  bool
	}{
		{
			name:       "authtype offered uses password when bearer is off",
			kind:       forge.KindGitea,
			host:       "gitea.example",
			capability: []string{credential.Authtype},
			want: []credential.Pair{
				{Key: credential.Capability, Value: credential.Authtype},
				{Key: credential.Password, Value: "access-token"},
				{Key: credential.Username, Value: "oauth2"},
				{Key: credential.OAuthRefreshToken, Value: "refresh-token"},
			},
		},
		{
			name:      "bearer without an offer stays on the password path",
			kind:      forge.KindGitea,
			host:      "gitea.example",
			useBearer: true,
			want: []credential.Pair{
				{Key: credential.Password, Value: "access-token"},
				{Key: credential.Username, Value: "oauth2"},
				{Key: credential.OAuthRefreshToken, Value: "refresh-token"},
			},
		},
		{
			name:       "offered bearer on gitea emits credential",
			kind:       forge.KindGitea,
			host:       "gitea.example",
			capability: []string{credential.Authtype},
			useBearer:  true,
			want: []credential.Pair{
				{Key: credential.Capability, Value: credential.Authtype},
				{Key: credential.Authtype, Value: bearerValue},
				{Key: credential.Credential, Value: "access-token"},
				{Key: credential.OAuthRefreshToken, Value: "refresh-token"},
			},
		},
		{
			name:       "offered bearer on github stays on password",
			kind:       forge.KindGitHub,
			host:       "github.com",
			capability: []string{credential.Authtype},
			useBearer:  true,
			want: []credential.Pair{
				{Key: credential.Capability, Value: credential.Authtype},
				{Key: credential.Password, Value: "access-token"},
				{Key: credential.Username, Value: "oauth2"},
				{Key: credential.OAuthRefreshToken, Value: "refresh-token"},
			},
		},
		{
			name:       "bitbucket default username is x-token-auth",
			kind:       forge.KindBitbucket,
			host:       "bitbucket.org",
			capability: []string{credential.Authtype},
			want: []credential.Pair{
				{Key: credential.Capability, Value: credential.Authtype},
				{Key: credential.Password, Value: "access-token"},
				{Key: credential.Username, Value: "x-token-auth"},
				{Key: credential.OAuthRefreshToken, Value: "refresh-token"},
			},
		},
		{
			name:     "existing username is not replaced",
			kind:     forge.KindGitLab,
			host:     "gitlab.com",
			username: "alice",
			want: []credential.Pair{
				{Key: credential.Password, Value: "access-token"},
				{Key: credential.OAuthRefreshToken, Value: "refresh-token"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := baseRequest()
			req.Username = tt.username
			req.Capability = tt.capability
			detected := detectedFor(tt.kind, tt.host)

			if tt.useBearer && detected.SupportsBearer() != (tt.kind == forge.KindGitea) {
				t.Fatalf("SupportsBearer() = %v", detected.SupportsBearer())
			}

			got := emit(req, detected, token, Options{UseBearer: tt.useBearer}, now, DefaultExpiryMargin)
			if !reflect.DeepEqual(tt.want, got.Pairs()) {
				t.Fatalf("pairs\nwant %#v\ngot  %#v", tt.want, got.Pairs())
			}
		})
	}
}

// TestEmitRefreshOmission drops an empty refresh token.
func TestEmitRefreshOmission(t *testing.T) {
	t.Parallel()

	token := tokenAt(time.Time{})
	token.AccessToken = "access-token"
	got := emit(
		baseRequest(),
		detectedFor(forge.KindGitHub, "github.com"),
		token,
		Options{},
		time.Now(),
		DefaultExpiryMargin,
	)
	if _, found := pairValue(got, credential.OAuthRefreshToken); found {
		t.Fatal("oauth_refresh_token emitted")
	}
}

// detectedFor builds a Detected value without a live registry.
func detectedFor(kind forge.Kind, host string) forge.Detected {
	return forge.Detected{
		Kind:  kind,
		Host:  host,
		Realm: "",
	}
}

// baseRequest is an https request with no username.
func baseRequest() credential.Request {
	return credential.Request{
		Protocol:          "https",
		Host:              "git.example",
		Path:              "",
		Username:          "",
		Password:          "",
		URL:               "",
		WWWAuth:           nil,
		Capability:        nil,
		OAuthRefreshToken: "",
		Extra:             nil,
	}
}

// tokenAt returns a token expiring at expiry.
func tokenAt(expiry time.Time) oauth.Token {
	return oauth.Token{
		AccessToken:  "access-token",
		TokenType:    bearerValue,
		RefreshToken: "",
		Expiry:       expiry,
	}
}

// pairValue returns the first value stored for key.
func pairValue(response credential.Response, key string) (string, bool) {
	for _, pair := range response.Pairs() {
		if pair.Key == key {
			return pair.Value, true
		}
	}

	return "", false
}
