// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDefaultUsernameIsTheBitbucketTokenNameOnlyForBitbucket covers the
// username placeholder.
//
// Bitbucket accepts an OAuth token over HTTPS only with the literal username
// x-token-auth. Every other forge ignores the username, and oauth2 is a
// placeholder rather than an account name.
func TestDefaultUsernameIsTheBitbucketTokenNameOnlyForBitbucket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		host string
		want string
	}{
		{host: "bitbucket.org", want: "x-token-auth"},
		{host: "github.com", want: "oauth2"},
		{host: "gitlab.com", want: "oauth2"},
		{host: "bitbucket.example.com", want: "oauth2"},
		{host: "", want: "oauth2"},
	}

	for _, tt := range tests {
		got := Detected{Host: tt.host, Kind: KindBitbucket}.DefaultUsername()
		assert.Equal(t, tt.want, got, "host %q", tt.host)
	}
}

// TestLoginHintParamFollowsTheFamily checks the account preselection key.
func TestLoginHintParamFollowsTheFamily(t *testing.T) {
	t.Parallel()

	want := map[Kind]string{
		KindUnknown:          "",
		KindGitHub:           "login",
		KindGitHubEnterprise: "login",
		KindGitLab:           "",
		KindGitea:            "",
		KindForgejo:          "",
		KindBitbucket:        "",
		KindGoogleSource:     "login_hint",
	}

	for kind, param := range want {
		assert.Equal(t, param, Detected{Kind: kind}.LoginHintParam(), "kind %s", kind)
	}
}

// TestSupportsBearer covers the three ways a host qualifies.
//
// The answer decides whether a token is sent as an Authorization: Bearer
// header, so a host that cannot take one must say no, or every request to it
// fails.
func TestSupportsBearer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		detected Detected
		want     bool
	}{
		{name: "bitbucket.org by host", detected: Detected{Host: "bitbucket.org"}, want: true},
		{name: "codeberg.org by host", detected: Detected{Host: "codeberg.org"}, want: true},
		{name: "gitea.com by host", detected: Detected{Host: "gitea.com"}, want: true},
		{name: "gitea realm", detected: Detected{Host: "x.example", Realm: "Gitea"}, want: true},
		{name: "forgejo realm", detected: Detected{Host: "x.example", Realm: "Forgejo"}, want: true},
		{name: "gitea kind", detected: Detected{Host: "x.example", Kind: KindGitea}, want: true},
		{name: "forgejo kind", detected: Detected{Host: "x.example", Kind: KindForgejo}, want: true},
		{name: "google source kind", detected: Detected{Host: "x.example", Kind: KindGoogleSource}, want: true},
		{name: "github", detected: Detected{Host: "github.com", Kind: KindGitHub}, want: false},
		{name: "gitlab", detected: Detected{Host: "gitlab.com", Kind: KindGitLab}, want: false},
		{name: "gitlab realm", detected: Detected{Host: "x.example", Realm: "GitLab"}, want: false},
		{name: "unknown", detected: Detected{Host: "x.example"}, want: false},
		{name: "realm compared exactly", detected: Detected{Host: "x.example", Realm: "gitea"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.detected.SupportsBearer())
		})
	}
}

// TestConfigKeyHintForEveryShape pins the three sentence shapes.
func TestConfigKeyHintForEveryShape(t *testing.T) {
	t.Parallel()

	const gitURL = "https://git.example.com"

	tests := []struct {
		want string
		kind Kind
	}{
		{
			kind: KindGitLab,
			want: "Set Git config key credential.https://git.example.com.oauthClientId.",
		},
		{
			kind: KindBitbucket,
			want: "Set Git config keys credential.https://git.example.com.oauthClientId" +
				" and credential.https://git.example.com.oauthClientSecret.",
		},
		{
			kind: KindForgejo,
			want: "Set Git config keys credential.https://git.example.com.oauthClientId," +
				" credential.https://git.example.com.oauthAuthURL," +
				" and credential.https://git.example.com.oauthTokenURL.",
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, Detected{Kind: tt.kind}.ConfigKeyHint(gitURL), "kind %s", tt.kind)
	}
}

// TestCredentialKeyKeepsThePortAndPath checks the key matches what Git looks up.
//
// Git matches credential.<url>.* keys against the remote, so a key built
// without the port would never match a remote that has one.
func TestCredentialKeyKeepsThePortAndPath(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		"credential.https://git.example.com:8443.oauthClientId",
		credentialKey("https://git.example.com:8443", "oauthClientId"))
}

// TestRegistrationHintUsesTheRequestedEnterpriseHost checks the form address.
//
// A GitHub Enterprise application is registered on the instance itself, so
// the hint must not send the operator to github.com.
func TestRegistrationHintUsesTheRequestedEnterpriseHost(t *testing.T) {
	t.Parallel()

	hint := Detected{Host: "github.example.com", Kind: KindGitHubEnterprise}.registrationHint()

	assert.Contains(t, hint, "https://github.example.com/settings/applications/new")
	assert.NotContains(t, hint, "https://github.com/")
}
