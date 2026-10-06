// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/oauth2"
)

// TestDetectUsesRegistryBeforePrefix checks the detection order.
//
// github.com and gitea.com both match a self-hosted subdomain prefix, so a
// registry hit has to be consulted first or the public hosts would be
// misclassified as GitHub Enterprise and Gitea.
func TestDetectUsesRegistryBeforePrefix(t *testing.T) {
	t.Parallel()

	registry := New()

	tests := []struct {
		name string
		host string
		want Kind
	}{
		{name: "github.com is not enterprise", host: "github.com", want: KindGitHub},
		{name: "gitea.com is not self-hosted", host: "gitea.com", want: KindGitea},
		{name: "codeberg.org is forgejo", host: "codeberg.org", want: KindForgejo},
		{name: "gitlab.com is gitlab", host: "gitlab.com", want: KindGitLab},
		{name: "bitbucket.org is bitbucket", host: "bitbucket.org", want: KindBitbucket},
		{
			name: "gist.github.com is github",
			host: "gist.github.com",
			want: KindGitHub,
		},
		{
			name: "google source host",
			host: "android.googlesource.com",
			want: KindGoogleSource,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Detect(registry, tt.host, "")
			assert.Equal(t, tt.want, got.Kind)
			assert.Equal(t, tt.host, got.Host)
		})
	}
}

// TestDetectPrefixesSelfHosted covers the subdomain heuristic.
func TestDetectPrefixesSelfHosted(t *testing.T) {
	t.Parallel()

	registry := New()

	tests := []struct {
		name string
		host string
		want Kind
	}{
		{name: "gitlab prefix", host: "gitlab.example.com", want: KindGitLab},
		{name: "gitea prefix", host: "gitea.example.com", want: KindGitea},
		{name: "forgejo prefix", host: "forgejo.example.com", want: KindForgejo},
		{name: "github prefix is enterprise", host: "github.acme.io", want: KindGitHubEnterprise},
		{name: "googlesource suffix", host: "code.googlesource.com", want: KindGoogleSource},
		{name: "unknown host", host: "git.example.org", want: KindUnknown},
		{name: "empty host", host: "", want: KindUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Detect(registry, tt.host, "")
			assert.Equal(t, tt.want, got.Kind)
		})
	}
}

// TestDetectRealmOnlyWhenHostIsUnknown covers realm as a later signal.
//
// A known host must keep its registry kind even when the server sends a
// different realm, or a GitHub host answering with realm="GitLab" would be
// treated as GitLab.
func TestDetectRealmOnlyWhenHostIsUnknown(t *testing.T) {
	t.Parallel()

	registry := New()

	tests := []struct {
		name    string
		host    string
		wwwAuth string
		wantRlm string
		want    Kind
	}{
		{
			name:    "registry beats realm",
			host:    "github.com",
			wwwAuth: `Basic realm="GitLab"`,
			want:    KindGitHub,
			wantRlm: "GitLab",
		},
		{
			name:    "realm classifies an unknown host",
			host:    "unknown.example.com",
			wwwAuth: `Basic realm="GitLab"`,
			want:    KindGitLab,
			wantRlm: "GitLab",
		},
		{
			name:    "gitea realm",
			host:    "unknown.example.com",
			wwwAuth: `Basic realm="Gitea"`,
			want:    KindGitea,
			wantRlm: "Gitea",
		},
		{
			name:    "forgejo realm",
			host:    "unknown.example.com",
			wwwAuth: `Bearer realm="Forgejo"`,
			want:    KindForgejo,
			wantRlm: "Forgejo",
		},
		{
			name:    "github realm on an unknown host is enterprise",
			host:    "unknown.example.com",
			wwwAuth: `Bearer realm="GitHub"`,
			want:    KindGitHubEnterprise,
			wantRlm: "GitHub",
		},
		{
			name:    "NotGitLab is not GitLab",
			host:    "unknown.example.com",
			wwwAuth: `Basic realm="NotGitLab"`,
			want:    KindUnknown,
			wantRlm: "NotGitLab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Detect(registry, tt.host, tt.wwwAuth)
			assert.Equal(t, tt.want, got.Kind)
			assert.Equal(t, tt.wantRlm, got.Realm)
		})
	}
}

// TestDetectJoinsMultipleChallenges checks every challenge is considered.
//
// A server may send a known forge realm in a later header value, and a helper
// that stopped at the first one would miss it.
func TestDetectJoinsMultipleChallenges(t *testing.T) {
	t.Parallel()

	registry := New()
	wwwAuth := `Basic realm="Unknown"` + "\n" + `Bearer realm="Gitea"`

	got := Detect(registry, "unknown.example.com", wwwAuth)
	assert.Equal(t, KindGitea, got.Kind, "a known realm wins over an earlier unknown one")
}

// TestDetectWithNilRegistry checks a nil registry is a miss, not a panic.
func TestDetectWithNilRegistry(t *testing.T) {
	t.Parallel()

	got := Detect(nil, "github.com", "")
	assert.Equal(t, KindGitHubEnterprise, got.Kind, "github prefix still applies")
	assert.Equal(t, "github.com", got.Host)
}

// TestDetectWithNoHeader checks a host with no challenge still classifies.
func TestDetectWithNoHeader(t *testing.T) {
	t.Parallel()

	got := Detect(New(), "github.com", "")
	assert.Equal(t, KindGitHub, got.Kind)
	assert.Empty(t, got.Realm)
}

// TestDeriveRejectsHostedKinds checks derivation is limited to self-hosted
// families.
//
// A public host has no remote root to derive from, and a public client must
// come from the registry instead.
func TestDeriveRejectsHostedKinds(t *testing.T) {
	t.Parallel()

	const gitURL = "https://example.com"

	tests := []struct {
		name string
		kind Kind
	}{
		{name: "unknown", kind: KindUnknown},
		{name: "github", kind: KindGitHub},
		{name: "bitbucket", kind: KindBitbucket},
		{name: "google source", kind: KindGoogleSource},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Derive(tt.kind, gitURL)
			require.ErrorIs(t, err, errNotSelfHosted)
		})
	}
}

// TestDeriveSelfHostedEndpoints covers every self-hosted family.
func TestDeriveSelfHostedEndpoints(t *testing.T) {
	t.Parallel()

	const gitURL = "https://git.example.com"

	tests := []struct {
		name       string
		authPath   string
		tokenPath  string
		wantScopes []string
		kind       Kind
		authStyle  oauth2.AuthStyle
		deviceFlow bool
	}{
		{
			name:       "gitea",
			kind:       KindGitea,
			authPath:   "/login/oauth/authorize",
			tokenPath:  "/login/oauth/access_token",
			authStyle:  oauth2.AuthStyleInParams,
			wantScopes: nil,
		},
		{
			name:      "forgejo uses colon scopes",
			kind:      KindForgejo,
			authPath:  "/login/oauth/authorize",
			tokenPath: "/login/oauth/access_token",
			authStyle: oauth2.AuthStyleInParams,
			wantScopes: []string{
				"read:repository",
				"write:repository",
			},
		},
		{
			name:       "gitlab",
			kind:       KindGitLab,
			authPath:   "/oauth/authorize",
			tokenPath:  "/oauth/token",
			deviceFlow: true,
			authStyle:  oauth2.AuthStyleAutoDetect,
			wantScopes: []string{"read_repository", "write_repository"},
		},
		{
			name:       "github enterprise",
			kind:       KindGitHubEnterprise,
			authPath:   "/login/oauth/authorize",
			tokenPath:  "/login/oauth/access_token",
			deviceFlow: true,
			authStyle:  oauth2.AuthStyleAutoDetect,
			wantScopes: []string{"repo", "gist", "workflow"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := Derive(tt.kind, gitURL)
			require.NoError(t, err)

			assert.Equal(t, "git.example.com", client.Host)
			assert.Equal(t, gitURL+tt.authPath, client.Endpoint.AuthURL)
			assert.Equal(t, gitURL+tt.tokenPath, client.Endpoint.TokenURL)
			assert.Equal(t, tt.authStyle, client.Endpoint.AuthStyle)
			assert.Equal(t, tt.deviceFlow, client.DeviceFlow)
			assert.Equal(t, tt.wantScopes, client.Scopes)
			assert.True(t, client.PKCE, "PKCE is mandatory for a public client")
		})
	}
}

// TestDeriveDeviceURLOnlyWhenSupported checks the device endpoint is set only
// where the grant works.
func TestDeriveDeviceURLOnlyWhenSupported(t *testing.T) {
	t.Parallel()

	const gitURL = "https://git.example.com"

	gitlab, err := Derive(KindGitLab, gitURL)
	require.NoError(t, err)
	assert.Equal(t, gitURL+"/oauth/authorize_device", gitlab.Endpoint.DeviceAuthURL)

	enterprise, err := Derive(KindGitHubEnterprise, gitURL)
	require.NoError(t, err)
	assert.Equal(t, gitURL+"/login/device/code", enterprise.Endpoint.DeviceAuthURL)

	for _, kind := range []Kind{KindGitea, KindForgejo} {
		client, deriveErr := Derive(kind, gitURL)
		require.NoError(t, deriveErr)
		assert.Empty(t, client.Endpoint.DeviceAuthURL, "%s has no device grant", kind)
	}
}

// TestDeriveRejectsMalformedURL checks a bad remote root is an error, not a
// process exit.
//
// This runs during a git operation, so it has to return rather than take the
// process down.
func TestDeriveRejectsMalformedURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		gitURL string
	}{
		{name: "empty", gitURL: ""},
		{name: "no scheme", gitURL: "git.example.com"},
		{name: "no host", gitURL: "https://"},
		{name: "relative", gitURL: "/repo.git"},
		{name: "scp-like", gitURL: "git@example.com:repo.git"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Derive(KindGitea, tt.gitURL)
			require.Error(t, err)
		})
	}
}

// TestDeriveTrimsTrailingSlash checks the joined path has no double slash.
func TestDeriveTrimsTrailingSlash(t *testing.T) {
	t.Parallel()

	client, err := Derive(KindGitea, "https://git.example.com/")
	require.NoError(t, err)

	assert.NotContains(t, client.Endpoint.AuthURL, "//login", "path is joined once")
	assert.Equal(t, "git.example.com", client.Host)
}

// TestDeriveKeepsPortInHost checks a non-default port is preserved.
func TestDeriveKeepsPortInHost(t *testing.T) {
	t.Parallel()

	client, err := Derive(KindGitea, "https://git.example.com:8443")
	require.NoError(t, err)

	assert.Equal(t, "git.example.com:8443", client.Host)
}

// TestDeriveUsesSharedGiteaClientID checks self-hosted Gitea and Forgejo use
// the application their own server registers, so an instance works before an
// operator registers anything.
func TestDeriveUsesSharedGiteaClientID(t *testing.T) {
	t.Parallel()

	gitea, err := Derive(KindGitea, "https://git.example.com")
	require.NoError(t, err)

	forgejo, err := Derive(KindForgejo, "https://git.example.com")
	require.NoError(t, err)

	assert.Equal(t, universalGiteaClientID, gitea.ClientID)
	assert.Equal(t, universalGiteaClientID, forgejo.ClientID)
	assert.Empty(t, gitea.ClientSecret, "a public client has no secret")
}

// TestDeriveSelfHostedNeedsNoClientID checks the operator is not blocked.
func TestDeriveSelfHostedNeedsNoClientID(t *testing.T) {
	t.Parallel()

	for _, kind := range []Kind{KindGitLab, KindGitHubEnterprise} {
		client, err := Derive(kind, "https://git.example.com")
		require.NoError(t, err)
		assert.Empty(t, client.ClientID, "%s has no built-in application", kind)
	}
}
