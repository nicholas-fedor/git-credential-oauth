// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package get_test exercises Service.Get against Mockery mocks.
package get_test

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"

	"github.com/nicholas-fedor/git-credential-oauth/internal/credential"
	"github.com/nicholas-fedor/git-credential-oauth/internal/forge"
	mockForge "github.com/nicholas-fedor/git-credential-oauth/internal/forge/mocks"
	"github.com/nicholas-fedor/git-credential-oauth/internal/get"
	mockGet "github.com/nicholas-fedor/git-credential-oauth/internal/get/mocks"
	"github.com/nicholas-fedor/git-credential-oauth/internal/oauth"
	mockOauth "github.com/nicholas-fedor/git-credential-oauth/internal/oauth/mocks"
)

// TestGetEmitsOrderedKeysClampAndAuthtype asserts wire order, margin clamp, and gating.
func TestGetEmitsOrderedKeysClampAndAuthtype(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.September, 30, 12, 0, 0, 0, time.UTC)
	outside := now.Add(time.Hour)
	inside := now.Add(30 * time.Second)
	outsideUnix := strconv.FormatInt(outside.Add(-get.DefaultExpiryMargin).Unix(), 10)
	nowUnix := strconv.FormatInt(now.Unix(), 10)

	tests := []struct {
		expiry     time.Time
		name       string
		host       string
		capability []string
		want       []credential.Pair
		kind       forge.Kind
		useBearer  bool
	}{
		{
			name:       "password path keeps capability first",
			host:       "git.example",
			kind:       forge.KindGitHub,
			capability: []string{credential.Authtype},
			expiry:     outside,
			want: []credential.Pair{
				{Key: credential.Capability, Value: credential.Authtype},
				{Key: credential.Password, Value: "access-token"},
				{Key: credential.Username, Value: "oauth2"},
				{Key: credential.PasswordExpiryUTC, Value: outsideUnix},
				{Key: credential.OAuthRefreshToken, Value: "refresh-token"},
			},
		},
		{
			name:   "missing authtype offer omits capability",
			host:   "git.example",
			kind:   forge.KindGitHub,
			expiry: outside,
			want: []credential.Pair{
				{Key: credential.Password, Value: "access-token"},
				{Key: credential.Username, Value: "oauth2"},
				{Key: credential.PasswordExpiryUTC, Value: outsideUnix},
				{Key: credential.OAuthRefreshToken, Value: "refresh-token"},
			},
		},
		{
			name:       "offered bearer emits credential before expiry",
			host:       "gitea.example",
			kind:       forge.KindGitea,
			capability: []string{credential.Authtype},
			useBearer:  true,
			expiry:     outside,
			want: []credential.Pair{
				{Key: credential.Capability, Value: credential.Authtype},
				{Key: credential.Authtype, Value: "Bearer"},
				{Key: credential.Credential, Value: "access-token"},
				{Key: credential.PasswordExpiryUTC, Value: outsideUnix},
				{Key: credential.OAuthRefreshToken, Value: "refresh-token"},
			},
		},
		{
			name:       "inside margin clamps expiry to now",
			host:       "git.example",
			kind:       forge.KindGitHub,
			capability: []string{credential.Authtype},
			expiry:     inside,
			want: []credential.Pair{
				{Key: credential.Capability, Value: credential.Authtype},
				{Key: credential.Password, Value: "access-token"},
				{Key: credential.Username, Value: "oauth2"},
				{Key: credential.PasswordExpiryUTC, Value: nowUnix},
				{Key: credential.OAuthRefreshToken, Value: "refresh-token"},
			},
		},
		{
			name:       "past expiry clamps to now",
			host:       "git.example",
			kind:       forge.KindGitHub,
			capability: []string{credential.Authtype},
			expiry:     now.Add(-time.Hour),
			want: []credential.Pair{
				{Key: credential.Capability, Value: credential.Authtype},
				{Key: credential.Password, Value: "access-token"},
				{Key: credential.Username, Value: "oauth2"},
				{Key: credential.PasswordExpiryUTC, Value: nowUnix},
				{Key: credential.OAuthRefreshToken, Value: "refresh-token"},
			},
		},
		{
			name:      "bearer without an offer does not emit capability",
			host:      "gitea.example",
			kind:      forge.KindGitea,
			useBearer: true,
			expiry:    time.Time{},
			want: []credential.Pair{
				{Key: credential.Password, Value: "access-token"},
				{Key: credential.Username, Value: "oauth2"},
				{Key: credential.OAuthRefreshToken, Value: "refresh-token"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			token := oauth.Token{
				AccessToken:  "access-token",
				TokenType:    "Bearer",
				RefreshToken: "refresh-token",
				Expiry:       tt.expiry,
			}
			auth := mockOauth.NewMockAcquirer(t)
			auth.EXPECT().
				Acquire(mock.Anything, mock.Anything, mock.Anything).
				Return(token, nil).
				Once()
			svc := get.Service{Deps: get.Deps{
				Config: unsetConfig(t, nil),
				Auth:   auth,
				Forges: hostRegistry(t, map[string]forge.Client{
					tt.host: completeClient(tt.host, tt.kind),
				}),
				Log: optionalLogger(t),
				Now: func() time.Time { return now },
			}}
			req := credential.Request{
				Protocol:          "https",
				Host:              tt.host,
				Capability:        tt.capability,
				OAuthRefreshToken: "refresh-token",
			}

			got, err := svc.Get(t.Context(), req, get.Options{UseBearer: tt.useBearer})
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(tt.want, got.Pairs()) {
				t.Fatalf("pairs\nwant %#v\ngot  %#v", tt.want, got.Pairs())
			}
		})
	}
}

// TestGetWrapsDeviceUnsupported preserves the device-flow sentinel.
func TestGetWrapsDeviceUnsupported(t *testing.T) {
	t.Parallel()

	auth := mockOauth.NewMockAcquirer(t)
	auth.EXPECT().
		Acquire(mock.Anything, mock.Anything, mock.Anything).
		Return(oauth.Token{}, oauth.ErrDeviceUnsupported).
		Once()
	svc := get.Service{Deps: get.Deps{
		Config: unsetConfig(t, nil),
		Auth:   auth,
		Forges: hostRegistry(t, map[string]forge.Client{
			"git.example": completeClient("git.example", forge.KindGitHub),
		}),
	}}

	_, err := svc.Get(t.Context(), credential.Request{
		Protocol: "https",
		Host:     "git.example",
	}, get.Options{Device: true})
	if !errors.Is(err, oauth.ErrDeviceUnsupported) {
		t.Fatalf("error %v", err)
	}
}

// TestGetConfigErrorIsNotUnset returns a config reader failure.
func TestGetConfigErrorIsNotUnset(t *testing.T) {
	t.Parallel()

	auth := mockOauth.NewMockAcquirer(t)
	svc := get.Service{Deps: get.Deps{
		Config: unsetConfig(t, errGitBroke),
		Auth:   auth,
		Forges: mockForge.NewMockRegistry(t),
	}}

	_, err := svc.Get(t.Context(), credential.Request{
		Protocol: "https",
		Host:     "git.example",
	}, get.Options{})
	if err == nil || !errors.Is(err, errGitBroke) {
		t.Fatalf("error %v", err)
	}

	auth.AssertNotCalled(t, "Acquire", mock.Anything, mock.Anything, mock.Anything)
}

// TestGetStopsAtTheHintWithoutAnApplication runs the built-in table as shipped.
//
// This program carries no application for these hosts, so an operator who has
// configured nothing must get the registration hint and no browser. Reaching
// Acquire would mean a client ID came from somewhere other than their config.
func TestGetStopsAtTheHintWithoutAnApplication(t *testing.T) {
	t.Parallel()

	hosts := []string{
		"github.com",
		"gist.github.com",
		"gitlab.com",
		"salsa.debian.org",
		"bitbucket.org",
		"android.googlesource.com",
	}

	for _, host := range hosts {
		t.Run(host, func(t *testing.T) {
			t.Parallel()

			auth := mockOauth.NewMockAcquirer(t)
			svc := get.Service{Deps: get.Deps{
				Config: unsetConfig(t, nil),
				Auth:   auth,
				Forges: forge.New(),
			}}

			_, err := svc.Get(t.Context(), credential.Request{
				Protocol: "https",
				Host:     host,
			}, get.Options{})
			if !errors.Is(err, get.ErrMissingConfig) {
				t.Fatalf("error %v, want missing configuration", err)
			}

			wantKey := "credential.https://" + host + ".oauthClientId"
			if !strings.Contains(err.Error(), wantKey) {
				t.Fatalf("hint %q does not name %s", err.Error(), wantKey)
			}

			auth.AssertNotCalled(t, "Acquire", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// TestGetUsesTheServerRegisteredApplication covers the two hosts that need no
// operator keys.
//
// Gitea and Forgejo register an application for this helper themselves, so
// these hosts start a grant with that ID and no secret.
func TestGetUsesTheServerRegisteredApplication(t *testing.T) {
	t.Parallel()

	const serverRegisteredID = "a4792ccc-144e-407e-86c9-5e7d8d9c3269"

	for _, host := range []string{"codeberg.org", "gitea.com"} {
		t.Run(host, func(t *testing.T) {
			t.Parallel()

			var got oauth2.Config

			auth := mockOauth.NewMockAcquirer(t)
			auth.EXPECT().
				Acquire(mock.Anything, mock.Anything, mock.Anything).
				Run(func(_ context.Context, cfg oauth2.Config, _ oauth.Input) {
					got = cfg
				}).
				Return(oauth.Token{AccessToken: "access-token"}, nil).
				Once()
			svc := get.Service{Deps: get.Deps{
				Config: unsetConfig(t, nil),
				Auth:   auth,
				Forges: forge.New(),
			}}

			_, err := svc.Get(t.Context(), credential.Request{
				Protocol: "https",
				Host:     host,
			}, get.Options{})
			if err != nil {
				t.Fatal(err)
			}

			if got.ClientID != serverRegisteredID || got.ClientSecret != "" {
				t.Fatalf("client %q with secret %q", got.ClientID, got.ClientSecret)
			}
		})
	}
}

// TestGetCompletesABuiltInHostFromOperatorKeys is the only way a hosted forge
// without a server-registered application can work.
//
// The built-in entry supplies the endpoints and scopes, and the operator's Git
// config supplies the application.
func TestGetCompletesABuiltInHostFromOperatorKeys(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"credential.oauthClientId":     "operator-id",
		"credential.oauthClientSecret": "operator-secret",
	}

	cfg := mockGet.NewMockConfigReader(t)
	cfg.EXPECT().
		Get(mock.Anything, mock.Anything, "https://github.com").
		RunAndReturn(func(_ context.Context, key, _ string) (string, bool, error) {
			value, found := values[key]

			return value, found, nil
		})

	var got oauth2.Config

	auth := mockOauth.NewMockAcquirer(t)
	auth.EXPECT().
		Acquire(mock.Anything, mock.Anything, mock.Anything).
		Run(func(_ context.Context, cfg oauth2.Config, _ oauth.Input) {
			got = cfg
		}).
		Return(oauth.Token{AccessToken: "access-token"}, nil).
		Once()
	svc := get.Service{Deps: get.Deps{
		Config: cfg,
		Auth:   auth,
		Forges: forge.New(),
	}}

	_, err := svc.Get(t.Context(), credential.Request{
		Protocol: "https",
		Host:     "github.com",
	}, get.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if got.ClientID != "operator-id" || got.ClientSecret != "operator-secret" {
		t.Fatalf("client %q with secret %q", got.ClientID, got.ClientSecret)
	}

	if got.Endpoint.TokenURL != "https://github.com/login/oauth/access_token" {
		t.Fatalf("token URL %q", got.Endpoint.TokenURL)
	}
}

func hostRegistry(t *testing.T, byHost map[string]forge.Client) *mockForge.MockRegistry {
	t.Helper()

	reg := mockForge.NewMockRegistry(t)
	reg.EXPECT().Lookup(mock.Anything).RunAndReturn(func(host string) (forge.Client, bool) {
		client, found := byHost[host]

		return client, found
	})
	reg.EXPECT().SelfHosted(mock.Anything).Return(false).Maybe()

	return reg
}

func unsetConfig(t *testing.T, err error) *mockGet.MockConfigReader {
	t.Helper()

	cfg := mockGet.NewMockConfigReader(t)
	cfg.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Return("", false, err)

	return cfg
}

func optionalLogger(t *testing.T) *mockGet.MockLogger {
	t.Helper()

	log := mockGet.NewMockLogger(t)
	log.EXPECT().Debug(mock.Anything, mock.Anything).Maybe()
	log.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything).Maybe()
	log.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	log.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything).Maybe()
	log.EXPECT().Warn(mock.Anything, mock.Anything).Maybe()
	log.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything).Maybe()
	log.EXPECT().Error(mock.Anything, mock.Anything).Maybe()
	log.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything).Maybe()

	return log
}

func completeClient(host string, kind forge.Kind) forge.Client {
	return forge.Client{
		Host:         host,
		Kind:         kind,
		ClientID:     "id-" + kind.String(),
		ClientSecret: "secret",
		Scopes:       []string{"repo"},
		Endpoint: oauth2.Endpoint{
			AuthURL:       "https://" + host + "/authorize",
			DeviceAuthURL: "",
			TokenURL:      "https://" + host + "/token",
			AuthStyle:     oauth2.AuthStyleInParams,
		},
		PKCE:       true,
		DeviceFlow: false,
	}
}

var errGitBroke = errors.New("git broke")
