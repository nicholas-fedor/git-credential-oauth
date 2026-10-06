// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/oauth2"

	"github.com/nicholas-fedor/git-credential-oauth/internal/credential"
	"github.com/nicholas-fedor/git-credential-oauth/internal/forge"
	"github.com/nicholas-fedor/git-credential-oauth/internal/oauth"
)

// TestResolveReferenceRejectsCleartext covers the endpoint scheme guard.
//
// A configured endpoint receives the authorization code and the access token.
// RFC 9700 section 2.6 forbids transmitting them over an unencrypted
// connection, so a cleartext endpoint must fail the request rather than send a
// token in the open.
func TestResolveReferenceRejectsCleartext(t *testing.T) {
	t.Parallel()

	const base = "https://git.example.com"

	tests := []struct {
		name string
		ref  string
	}{
		{name: "absolute http", ref: "http://evil.example.com/token"},
		{name: "absolute http on the git host", ref: "http://git.example.com/token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolved, err := resolveReference(base, tt.ref)
			if err == nil {
				t.Fatalf("resolveReference(%q) = %q, want an insecure error", tt.ref, resolved)
			}

			require.ErrorIs(t, err, errInsecureEndpoint)
		})
	}
}

// TestResolveReferenceAcceptsHTTPS covers the ordinary cases still resolve.
func TestResolveReferenceAcceptsHTTPS(t *testing.T) {
	t.Parallel()

	const base = "https://git.example.com"

	tests := []struct {
		name string
		ref  string
		want string
	}{
		{
			name: "relative joins the base",
			ref:  "/oauth/token",
			want: "https://git.example.com/oauth/token",
		},
		{
			name: "absolute is unchanged",
			ref:  "https://sso.example.com/oauth/token",
			want: "https://sso.example.com/oauth/token",
		},
		{
			name: "relative without a leading slash",
			ref:  "oauth/token",
			want: "https://git.example.com/oauth/token",
		},
		{
			name: "scheme-relative inherits the base scheme",
			ref:  "//sso.example.com/oauth/token",
			want: "https://sso.example.com/oauth/token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveReference(base, tt.ref)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestResolveConfiguredURLClearsOnEmpty checks an empty override stays empty.
//
// An operator who empties a key is asking for the derived default back, not
// for the key to resolve to the base URL.
func TestResolveConfiguredURLClearsOnEmpty(t *testing.T) {
	t.Parallel()

	got, err := resolveConfiguredURL("https://git.example.com", "")
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestParsedMarginSubstitutesDefaultWhenUnset covers the unset path.
//
// An absent key is usable: the default applies and there is nothing to warn
// about, so the request continues without a diagnostic.
func TestParsedMarginSubstitutesDefaultWhenUnset(t *testing.T) {
	t.Parallel()

	margin, usable := parsedMargin(configOverrides{})
	assert.Equal(t, DefaultExpiryMargin, margin)
	assert.True(t, usable, "an absent key is not a configuration error")
}

// TestParsedMarginUsesConfiguredValue covers a valid configured margin.
func TestParsedMarginUsesConfiguredValue(t *testing.T) {
	t.Parallel()

	overrides := configOverrides{
		ExpiryMargin:      "5m",
		ExpiryMarginFound: true,
	}

	margin, usable := parsedMargin(overrides)
	assert.Equal(t, 5*time.Minute, margin)
	assert.True(t, usable, "a parsed value is usable")
}

// TestParsedMarginFallsBackOnInvalidValue covers a typo.
//
// A bad duration must not lock a user out of their credential, so the default
// is substituted and the request continues.
func TestParsedMarginFallsBackOnInvalidValue(t *testing.T) {
	t.Parallel()

	overrides := configOverrides{
		ExpiryMargin:      "not-a-duration",
		ExpiryMarginFound: true,
	}

	margin, usable := parsedMargin(overrides)
	assert.Equal(t, DefaultExpiryMargin, margin)
	assert.False(t, usable, "a set but unparseable value is the one unusable case")
}

// TestParseExpiryMarginAcceptsZero covers a deliberate zero margin.
func TestParseExpiryMarginAcceptsZero(t *testing.T) {
	t.Parallel()

	margin, ok := parseExpiryMargin("0s")
	assert.True(t, ok)
	assert.Equal(t, time.Duration(0), margin)
}

// TestParseExpiryMarginAcceptsNegative covers an operator clamping harder.
func TestParseExpiryMarginAcceptsNegative(t *testing.T) {
	t.Parallel()

	margin, ok := parseExpiryMargin("-30s")
	assert.True(t, ok)
	assert.Equal(t, -30*time.Second, margin)
}

// TestConfigIncomplete covers the completeness check for a client.
func TestConfigIncomplete(t *testing.T) {
	t.Parallel()

	complete := forge.Client{
		Host:     "git.example.com",
		ClientID: "id",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://git.example.com/authorize",
			TokenURL: "https://git.example.com/token",
		},
	}

	assert.False(t, configIncomplete(complete))

	noID := complete
	noID.ClientID = ""
	assert.True(t, configIncomplete(noID), "a client without an ID is incomplete")

	noAuth := complete
	noAuth.Endpoint.AuthURL = ""
	assert.True(t, configIncomplete(noAuth), "a client without an auth URL is incomplete")

	noToken := complete
	noToken.Endpoint.TokenURL = ""
	assert.True(t, configIncomplete(noToken), "a client without a token URL is incomplete")
}

// TestMissingConfigHintRegistersBeforeNamingKeys covers a client with no ID.
//
// No application ships with this program, so a host with no client ID has
// nothing behind it yet. Naming a key alone would leave a first-time user with
// no idea where its value comes from.
func TestMissingConfigHintRegistersBeforeNamingKeys(t *testing.T) {
	t.Parallel()

	const gitURL = "https://github.com"

	detected := forge.Detected{Host: "github.com", Kind: forge.KindGitHub}
	client := forge.Client{
		Host: "github.com",
		Kind: forge.KindGitHub,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
	}

	hint := missingConfigHint(detected, client, gitURL)

	assert.Equal(t, detected.SetupHint(gitURL), hint)
	assert.Contains(t, hint, "Register an OAuth application")
	assert.Contains(t, hint, "credential.https://github.com.oauthClientId")
	assert.Contains(t, hint, "credential.https://github.com.oauthClientSecret")
}

// TestMissingConfigHintNamesEndpointsOnceAnIDIsSet covers the second step.
//
// An operator who has already set a client ID has registered an application,
// so telling them to register one again would be noise. What they still lack
// is the endpoints.
func TestMissingConfigHintNamesEndpointsOnceAnIDIsSet(t *testing.T) {
	t.Parallel()

	const gitURL = "https://git.example.com"

	detected := forge.Detected{Host: "git.example.com", Kind: forge.KindUnknown}
	client := forge.Client{Host: "git.example.com", ClientID: "operator-id"}

	hint := missingConfigHint(detected, client, gitURL)

	assert.Equal(
		t,
		"Set Git config key credential.https://git.example.com.oauthClientId."+
			" Also set oauthAuthURL and oauthTokenURL.",
		hint,
	)
	assert.NotContains(t, hint, "Register")
}

// canceledContext returns a context that is already done.
//
// Resolving a self-hosted client fetches the server's metadata document. A
// canceled context makes that fetch fail before it leaves the process, so the
// tests exercise the fallback without any network access.
func canceledContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	return ctx
}

// TestResolveRequiresARegistry fails rather than panics.
func TestResolveRequiresARegistry(t *testing.T) {
	t.Parallel()

	_, err := resolve(t.Context(), nil, credential.Request{Host: "x"}, Options{}, configOverrides{})
	require.ErrorIs(t, err, errNilRegistry)
}

// TestResolveCompletesABuiltInHostFromOverrides runs the whole resolution.
func TestResolveCompletesABuiltInHostFromOverrides(t *testing.T) {
	t.Parallel()

	got, err := resolve(t.Context(), forge.New(), credential.Request{
		Protocol:          "https",
		Host:              "github.com",
		Username:          "octocat",
		OAuthRefreshToken: "refresh",
	}, Options{Device: true}, configOverrides{
		ClientID:          "id",
		ClientIDFound:     true,
		ClientSecret:      "secret",
		ClientSecretFound: true,
		RedirectURL:       "http://127.0.0.1:7171/cb",
		RedirectURLFound:  true,
	})
	require.NoError(t, err)

	assert.Equal(t, "https://github.com", got.GitURL)
	assert.Equal(t, forge.KindGitHub, got.Detected.Kind)
	assert.Equal(t, "id", got.Config.ClientID)
	assert.Equal(t, "secret", got.Config.ClientSecret)
	assert.Equal(t, "http://127.0.0.1:7171/cb", got.Config.RedirectURL)
	assert.Equal(t, []string{"repo", "gist", "workflow"}, got.Config.Scopes)
	assert.Equal(t, oauth.Input{
		RefreshToken:  "refresh",
		AuthURLSuffix: "&login=octocat",
		Device:        true,
		DeviceFlow:    true,
		PKCE:          true,
	}, got.Input)
	assert.Equal(t, DefaultExpiryMargin, got.Margin)
}

// TestResolveReportsMissingConfigWithItsFindings checks a partial result.
//
// The caller logs warnings even when the request fails, so the partial
// resolution must still carry the margin and scopes it found.
func TestResolveReportsMissingConfigWithItsFindings(t *testing.T) {
	t.Parallel()

	got, err := resolve(t.Context(), forge.New(), credential.Request{
		Protocol: "https",
		Host:     "gitlab.com",
	}, Options{}, configOverrides{ExpiryMargin: "bad", ExpiryMarginFound: true})

	var missing *MissingConfigError
	require.ErrorAs(t, err, &missing)
	assert.Equal(t, "gitlab.com", missing.Host)
	assert.Equal(t, "https://gitlab.com", missing.GitURL)
	assert.Contains(t, missing.Hint, "https://gitlab.com/-/user_settings/applications")

	assert.True(t, got.MarginInvalid)
	assert.Equal(t, []string{"read_repository", "write_repository"}, got.Scopes)
	assert.Empty(t, got.Config.ClientID, "no grant configuration is built")
}

// TestResolveRejectsACleartextEndpointOverride checks the guard reaches resolve.
func TestResolveRejectsACleartextEndpointOverride(t *testing.T) {
	t.Parallel()

	_, err := resolve(t.Context(), forge.New(), credential.Request{
		Protocol: "https",
		Host:     "github.com",
	}, Options{}, configOverrides{
		ClientID:      "id",
		ClientIDFound: true,
		TokenURL:      "http://github.com/token",
		TokenURLFound: true,
	})
	require.ErrorIs(t, err, errInsecureEndpoint)
}

// TestResolveRejectsACleartextSelfHostedRemote covers a derived host.
func TestResolveRejectsACleartextSelfHostedRemote(t *testing.T) {
	t.Parallel()

	_, err := resolve(canceledContext(t), forge.New(), credential.Request{
		Protocol: "http",
		Host:     "gitea.example.invalid",
	}, Options{}, configOverrides{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "derive oauth client")
}

// TestResolveDerivesASelfHostedGiteaWithoutConfiguration covers the no-setup case.
func TestResolveDerivesASelfHostedGiteaWithoutConfiguration(t *testing.T) {
	t.Parallel()

	got, err := resolve(canceledContext(t), forge.New(), credential.Request{
		Protocol: "https",
		Host:     "gitea.example.invalid",
	}, Options{}, configOverrides{})
	require.NoError(t, err)

	assert.Equal(t, forge.KindGitea, got.Detected.Kind)
	assert.NotEmpty(t, got.Config.ClientID)
	assert.Equal(t, "https://gitea.example.invalid/login/oauth/access_token", got.Config.Endpoint.TokenURL)
}

// TestLookupOrDerive covers the three outcomes.
func TestLookupOrDerive(t *testing.T) {
	t.Parallel()

	reg := forge.New()

	client, registered, err := lookupOrDerive(reg, forge.Detected{Kind: forge.KindGitHub}, "github.com", "https://github.com")
	require.NoError(t, err)
	assert.True(t, registered)
	assert.Equal(t, "github.com", client.Host)

	client, registered, err = lookupOrDerive(
		reg, forge.Detected{Kind: forge.KindGitLab}, "gitlab.example.com", "https://gitlab.example.com",
	)
	require.NoError(t, err)
	assert.False(t, registered)
	assert.Equal(t, "https://gitlab.example.com/oauth/token", client.Endpoint.TokenURL)

	client, registered, err = lookupOrDerive(
		reg, forge.Detected{Kind: forge.KindUnknown}, "git.example.com", "https://git.example.com",
	)
	require.NoError(t, err)
	assert.False(t, registered)
	assert.Equal(t, forge.Client{}, client, "an unknown host gets no client")

	_, _, err = lookupOrDerive(reg, forge.Detected{Kind: forge.KindGitLab}, "gitlab.example.com", "http://gitlab.example.com")
	require.Error(t, err)
}

// TestDiscoverEndpointsFallsBackWhenTheFetchFails keeps the derived client.
//
// A host that publishes nothing, or cannot be reached, must not stop a
// credential that would otherwise work.
func TestDiscoverEndpointsFallsBackWhenTheFetchFails(t *testing.T) {
	t.Parallel()

	derived, err := forge.Derive(forge.KindGitLab, "https://gitlab.example.invalid")
	require.NoError(t, err)

	got := discoverEndpoints(canceledContext(t), forge.KindGitLab, "gitlab.example.invalid", derived)
	assert.Equal(t, derived, got)
}

// TestApplyOverridesReplacesOnlyWhatWasFound checks unset keys keep defaults.
func TestApplyOverridesReplacesOnlyWhatWasFound(t *testing.T) {
	t.Parallel()

	base := forge.Client{
		ClientID:     "default-id",
		ClientSecret: "",
		Scopes:       []string{"repo"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://git.example.com/authorize",
			TokenURL: "https://git.example.com/token",
		},
	}

	untouched, err := applyOverrides(base, "https://git.example.com", configOverrides{})
	require.NoError(t, err)
	assert.Equal(t, base, untouched.Client)
	assert.Empty(t, untouched.RedirectURL)

	replaced, err := applyOverrides(base, "https://git.example.com", configOverrides{
		ClientID:          "id",
		ClientIDFound:     true,
		ClientSecret:      "secret",
		ClientSecretFound: true,
		Scopes:            "  read   write ",
		ScopesFound:       true,
		RedirectURL:       "http://127.0.0.1:1/cb",
		RedirectURLFound:  true,
	})
	require.NoError(t, err)
	assert.Equal(t, "id", replaced.Client.ClientID)
	assert.Equal(t, "secret", replaced.Client.ClientSecret)
	assert.Equal(t, []string{"read", "write"}, replaced.Client.Scopes, "scopes split on whitespace")
	assert.Equal(t, "http://127.0.0.1:1/cb", replaced.RedirectURL)
	assert.Equal(t, []string{"repo"}, base.Scopes, "the base client is not modified")

	cleared, err := applyOverrides(base, "https://git.example.com", configOverrides{ClientIDFound: true})
	require.NoError(t, err)
	assert.Empty(t, cleared.Client.ClientID, "an empty value clears the default")
}

// TestApplyEndpointOverrides covers each endpoint key.
func TestApplyEndpointOverrides(t *testing.T) {
	t.Parallel()

	const gitURL = "https://git.example.com"

	base := forge.Client{Endpoint: oauth2.Endpoint{
		AuthURL:       "https://git.example.com/a",
		TokenURL:      "https://git.example.com/t",
		DeviceAuthURL: "https://git.example.com/d",
		AuthStyle:     oauth2.AuthStyleInParams,
	}}

	got, err := applyEndpointOverrides(base, gitURL, configOverrides{
		AuthURL:            "/oauth/authorize",
		AuthURLFound:       true,
		TokenURL:           "https://sso.example.com/token",
		TokenURLFound:      true,
		DeviceAuthURL:      "",
		DeviceAuthURLFound: true,
	})
	require.NoError(t, err)

	assert.Equal(t, "https://git.example.com/oauth/authorize", got.Endpoint.AuthURL, "relative to the remote")
	assert.Equal(t, "https://sso.example.com/token", got.Endpoint.TokenURL, "absolute is kept")
	assert.Empty(t, got.Endpoint.DeviceAuthURL, "an empty value clears the device endpoint")
	assert.Equal(t, oauth2.AuthStyleInParams, got.Endpoint.AuthStyle)

	for name, overrides := range map[string]configOverrides{
		"auth":   {AuthURL: "http://evil.example.com/a", AuthURLFound: true},
		"token":  {TokenURL: "http://evil.example.com/t", TokenURLFound: true},
		"device": {DeviceAuthURL: "http://evil.example.com/d", DeviceAuthURLFound: true},
	} {
		_, err = applyEndpointOverrides(base, gitURL, overrides)
		require.ErrorIs(t, err, errInsecureEndpoint, "%s endpoint", name)
	}
}

// TestResolveReferenceReportsUnparseableInput covers both parse failures.
func TestResolveReferenceReportsUnparseableInput(t *testing.T) {
	t.Parallel()

	_, err := resolveReference("https://git.example.com", "https://x/%zz")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse oauth url")

	_, err = resolveReference("://bad", "/token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse git url")
}

// TestResolveReferenceRejectsAMalformedResult covers fuzzer findings.
//
// Each of these is https, but would fail later inside the OAuth library with a
// less useful message.
func TestResolveReferenceRejectsAMalformedResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		base string
		ref  string
	}{
		{base: "https:", ref: "//::"},
		{base: "https:", ref: "/token"},
		{base: "https://git.example.com", ref: "https:token"},
	}

	for _, tt := range tests {
		_, err := resolveReference(tt.base, tt.ref)
		require.ErrorIs(t, err, errMalformedEndpoint, "base %q ref %q", tt.base, tt.ref)
	}
}

// TestProtocolURLDefaultsToHTTPS covers a request without a protocol.
func TestProtocolURLDefaultsToHTTPS(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "https://git.example.com", protocolURL("", "git.example.com"))
	assert.Equal(t, "http://git.example.com", protocolURL("http", "git.example.com"))
}

// TestEnsureMentionsAppendsOnlyMissingKeys covers the hint suffix.
func TestEnsureMentionsAppendsOnlyMissingKeys(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Set a.", ensureMentions("Set a.", "a"))
	assert.Equal(t, "Set a. Also set b.", ensureMentions("Set a.", "a", "b"))
	assert.Equal(t, "Hint. Also set b and c.", ensureMentions("Hint.", "b", "c"))
}

// TestJoinKeys covers one, two, and more names.
func TestJoinKeys(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "a", joinKeys([]string{"a"}))
	assert.Equal(t, "a and b", joinKeys([]string{"a", "b"}))
	assert.Equal(t, "a, b, c", joinKeys([]string{"a", "b", "c"}))
	assert.Empty(t, joinKeys(nil))
}

// TestAuthURLSuffix covers when a login hint is sent.
//
// The oauth2 placeholder is this program's own username, not an account, so
// sending it would preselect an account that does not exist.
func TestAuthURLSuffix(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "&login=octocat", authURLSuffix("octocat", "login"))
	assert.Empty(t, authURLSuffix("", "login"))
	assert.Empty(t, authURLSuffix("oauth2", "login"))
	assert.Empty(t, authURLSuffix("octocat", ""))
}

// TestOAuthConfigCopiesTheScopes checks the config does not share the slice.
func TestOAuthConfigCopiesTheScopes(t *testing.T) {
	t.Parallel()

	client := forge.Client{ClientID: "id", Scopes: []string{"a"}}
	cfg := oauthConfig(client, "http://127.0.0.1:1")
	cfg.Scopes[0] = "changed"

	assert.Equal(t, "a", client.Scopes[0])
	assert.Equal(t, "http://127.0.0.1:1", cfg.RedirectURL)
}

// TestOffendingScopesOnlyFlagsGiteaFamilies covers the scope-shape warning.
func TestOffendingScopesOnlyFlagsGiteaFamilies(t *testing.T) {
	t.Parallel()

	scopes := []string{"read_repository", "write:repository", "x-write_repository"}

	assert.Equal(t, []string{"read_repository", "x-write_repository"}, offendingScopes(forge.KindForgejo, scopes))
	assert.Equal(t, []string{"read_repository", "x-write_repository"}, offendingScopes(forge.KindGitea, scopes))
	assert.Nil(t, offendingScopes(forge.KindGitLab, scopes))
	assert.Nil(t, offendingScopes(forge.KindForgejo, []string{"read:repository"}))
}
