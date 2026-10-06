// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"net/url"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/oauth2"
)

// TestNewRegistersEveryBuiltinHost checks the built-in table is complete.
//
// builtinHostCount is a compile-time size the code maintains, so a host added
// to the table without bumping it would silently be missing.
func TestNewRegistersEveryBuiltinHost(t *testing.T) {
	t.Parallel()

	table := New()
	require.NotNil(t, table)

	hosts := table.Hosts()
	assert.Len(t, hosts, builtinHostCount, "host count matches the declared constant")
	assert.True(t, slices.IsSorted(hosts), "host list is sorted")
	assert.NotContains(t, hosts, "", "no empty host name")
}

// TestNewIsHTTPSEverywhere checks no public endpoint uses cleartext.
//
// RFC 9700 section 2.6 forbids transmitting an authorization response over an
// unencrypted connection, so a built-in host with an http endpoint would leak a
// token for every user.
func TestNewIsHTTPSEverywhere(t *testing.T) {
	t.Parallel()

	table := New()

	for _, host := range table.Hosts() {
		client, ok := table.Lookup(host)
		require.True(t, ok, "host %q must be registered", host)

		for name, endpoint := range map[string]string{
			"auth URL":   client.Endpoint.AuthURL,
			"token URL":  client.Endpoint.TokenURL,
			"device URL": client.Endpoint.DeviceAuthURL,
		} {
			if endpoint == "" {
				continue
			}

			parsed, err := url.Parse(endpoint)
			require.NoError(t, err, "host %q has an unparseable %s", host, name)
			assert.Equal(t, "https", parsed.Scheme, "host %q %s must be https", host, name)
		}
	}
}

// serverRegisteredHosts are the built-in hosts that carry a client ID.
//
// Gitea and Forgejo register an application for this helper themselves, so
// these two work before an operator registers anything. Every other host
// carries endpoints and scopes only. The set is named here rather than derived,
// so a host that gains an application by accident fails the assertions below.
var serverRegisteredHosts = []string{
	"codeberg.org",
	"gitea.com",
}

// gitHubHosts are the public GitHub hosts in the built-in table.
var gitHubHosts = []string{
	"github.com",
	"gist.github.com",
}

// TestNewShipsNoApplicationOfItsOwn checks which hosts carry a client ID.
//
// This program presents itself as no one else's application, so the only ID it
// may carry is the one a forge's own server registers. Every other host has to
// stop at the Git config hint until the operator supplies their application.
func TestNewShipsNoApplicationOfItsOwn(t *testing.T) {
	t.Parallel()

	table := New()

	for _, host := range table.Hosts() {
		client, ok := table.Lookup(host)
		require.True(t, ok)

		assert.Empty(t, client.ClientSecret, "host %q must publish no client secret", host)
		assert.NotEmpty(t, client.Endpoint.AuthURL, "host %q needs an auth URL", host)
		assert.NotEmpty(t, client.Endpoint.TokenURL, "host %q needs a token URL", host)
		assert.Equal(t, host, client.Host, "client Host matches its key")

		if slices.Contains(serverRegisteredHosts, host) {
			assert.Equal(t, universalGiteaClientID, client.ClientID,
				"host %q uses the application its server registers", host)

			continue
		}

		assert.Empty(t, client.ClientID, "host %q must publish no client ID", host)
	}
}

// TestServerRegisteredHostsAreGiteaLayout checks why those hosts have an ID.
//
// The shared ID only exists on servers that register it, which are the Gitea
// and Forgejo families. Giving it to any other family would send a client ID
// the forge has never heard of.
func TestServerRegisteredHostsAreGiteaLayout(t *testing.T) {
	t.Parallel()

	table := New()

	for _, host := range serverRegisteredHosts {
		client, ok := table.Lookup(host)
		require.True(t, ok, "host %q must be registered", host)
		assert.True(t, client.Kind.UsesPublicClientID(), "host %q family", host)
	}
}

// TestNewEnablesPKCEForEveryHost checks the S256 requirement holds broadly.
//
// This is a public client with no secret to protect, so RFC 9700 section 2.1.1
// requires PKCE on every host rather than only the self-hosted ones.
func TestNewEnablesPKCEForEveryHost(t *testing.T) {
	t.Parallel()

	table := New()

	for _, host := range table.Hosts() {
		client, ok := table.Lookup(host)
		require.True(t, ok)
		assert.True(t, client.PKCE, "host %q must use PKCE", host)
	}
}

// TestLookupIsCaseSensitive documents how host names are compared.
//
// A registry hit must beat the subdomain-prefix path, so github.com and
// gitea.com cannot be treated as self-hosted.
func TestLookupIsCaseSensitive(t *testing.T) {
	t.Parallel()

	table := New()

	_, ok := table.Lookup("github.com")
	assert.True(t, ok, "exact host matches")

	_, ok = table.Lookup("GitHub.com")
	assert.False(t, ok, "host comparison is case-sensitive")

	_, ok = table.Lookup("unknown.example.com")
	assert.False(t, ok, "unregistered host misses")
}

// TestLookupIsolatesScopes checks a caller cannot mutate the registry.
//
// Returning the stored client directly would let one request's scope rewrite
// every later request for that host.
func TestLookupIsolatesScopes(t *testing.T) {
	t.Parallel()

	table := New()

	first, ok := table.Lookup("github.com")
	require.True(t, ok)
	require.NotEmpty(t, first.Scopes)

	first.Scopes[0] = "tampered"

	second, ok := table.Lookup("github.com")
	require.True(t, ok)
	assert.NotEqual(t, "tampered", second.Scopes[0], "registry scope survived")
}

// TestNilTableIsSafe checks a nil registry degrades to a miss.
func TestNilTableIsSafe(t *testing.T) {
	t.Parallel()

	var table *Table

	client, ok := table.Lookup("github.com")
	assert.False(t, ok)
	assert.Equal(t, Client{}, client)
	assert.Nil(t, table.Hosts())
}

// TestWithDoesNotMutateReceiver checks With returns a new table.
func TestWithDoesNotMutateReceiver(t *testing.T) {
	t.Parallel()

	base := New()
	before := len(base.Hosts())

	extended := base.With(Client{
		Host:     "git.example.com",
		Kind:     KindGitea,
		ClientID: "added",
		Scopes:   []string{"read:repository"},
	})

	assert.Len(t, base.Hosts(), before, "receiver unchanged")
	require.Len(t, extended.Hosts(), before+1)
	assert.Contains(t, extended.Hosts(), "git.example.com")
}

// TestWithReplacesExistingHost checks a second With wins for one host.
func TestWithReplacesExistingHost(t *testing.T) {
	t.Parallel()

	base := New()
	replaced := base.With(Client{
		Host:     "github.com",
		Kind:     KindGitHub,
		ClientID: "replacement",
	})

	client, ok := replaced.Lookup("github.com")
	require.True(t, ok)
	assert.Equal(t, "replacement", client.ClientID)
	assert.Len(t, replaced.Hosts(), len(base.Hosts()), "no duplicate host")
}

// TestSelfHostedMatchesKind covers the registry classification.
func TestSelfHostedMatchesKind(t *testing.T) {
	t.Parallel()

	table := New()

	selfHosted := []Kind{KindGitHubEnterprise, KindGitLab, KindGitea, KindForgejo}
	hosted := []Kind{KindUnknown, KindGitHub, KindBitbucket, KindGoogleSource}

	for _, kind := range selfHosted {
		assert.True(t, table.SelfHosted(kind), "%s is self-hosted", kind)
		assert.True(t, kind.IsSelfHosted(), "%s IsSelfHosted", kind)
	}

	for _, kind := range hosted {
		assert.False(t, table.SelfHosted(kind), "%s is not self-hosted", kind)
		assert.False(t, kind.IsSelfHosted(), "%s IsSelfHosted", kind)
	}
}

// TestGitHubClientWaitsForOperatorKeys checks the GitHub entries are
// deliberately unusable until an operator supplies their own application.
//
// Leaving both fields empty forces the request to stop with the Git config
// hint instead, which is the only guidance a first-time user gets.
func TestGitHubClientWaitsForOperatorKeys(t *testing.T) {
	t.Parallel()

	table := New()

	for _, host := range gitHubHosts {
		client, ok := table.Lookup(host)
		require.True(t, ok, "host %q must be registered", host)

		assert.Equal(t, KindGitHub, client.Kind, "host %q family", host)
		assert.Empty(t, client.ClientID, "host %q must publish no client ID", host)
		assert.Empty(t, client.ClientSecret, "host %q must publish no client secret", host)

		// The endpoints are what make the host recognizable at all, so they
		// must still be present or the failure would be unexplained.
		assert.NotEmpty(t, client.Endpoint.AuthURL, "host %q needs an auth URL", host)
		assert.NotEmpty(t, client.Endpoint.TokenURL, "host %q needs a token URL", host)
		assert.True(t, client.DeviceFlow, "host %q must support the device grant", host)
	}
}

// TestGitHubHintNamesBothKeys checks the recovery path is actionable.
//
// The hint is the whole error a user sees, so it has to name the exact Git
// config keys for the host that failed rather than describing the problem.
// GitHub rejects the browser grant without the secret, so naming the client ID
// alone would send a user through the consent screen into a failed exchange.
func TestGitHubHintNamesBothKeys(t *testing.T) {
	t.Parallel()

	const gitURL = "https://github.com"

	detected := Detect(New(), "github.com", "")

	assert.Equal(t, KindGitHub, detected.Kind)
	assert.Equal(
		t,
		"Set Git config keys credential.https://github.com.oauthClientId"+
			" and credential.https://github.com.oauthClientSecret.",
		detected.ConfigKeyHint(gitURL),
	)
}

// TestGistHintRegistersOnGitHub checks a gist is sent to the right form.
//
// gist.github.com has no application settings of its own, so the form is on
// github.com while the keys still belong to the host that was asked.
func TestGistHintRegistersOnGitHub(t *testing.T) {
	t.Parallel()

	detected := Detect(New(), "gist.github.com", "")
	hint := detected.SetupHint("https://gist.github.com")

	assert.Contains(t, hint, "https://github.com/settings/applications/new")
	assert.Contains(t, hint, "credential.https://gist.github.com.oauthClientId")
}

// TestKindStringNamesEveryConstant checks the checksum linter's contract.
//
// String has to cover every declared constant, so a new kind that is forgotten
// shows up as a numeric name in a log.
func TestKindStringNamesEveryConstant(t *testing.T) {
	t.Parallel()

	names := map[string]bool{}

	for kind := KindUnknown; int(kind) < kindCount; kind++ {
		name := kind.String()
		assert.NotEmpty(t, name, "kind %d has no name", kind)
		assert.Equal(t, "Kind", name[:4], "kind %d name shape", kind)
		assert.False(t, names[name], "duplicate name %q", name)
		names[name] = true
	}
}

// TestKindStringForUndeclaredValue checks an out-of-range kind is visible.
func TestKindStringForUndeclaredValue(t *testing.T) {
	t.Parallel()

	negative := Kind(-1)
	assert.Contains(t, negative.String(), "-1")

	tooLarge := Kind(kindCount)
	assert.Contains(t, tooLarge.String(), "8")
}

// TestKindSumBoundsEveryConstant checks the checksum helper is a real bound.
func TestKindSumBoundsEveryConstant(t *testing.T) {
	t.Parallel()

	sum := KindUnknown.Sum()
	assert.Equal(t, kindCount, sum)

	for kind := KindUnknown; int(kind) < kindCount; kind++ {
		assert.Less(t, int(kind), sum, "kind %s is inside the bound", kind)
	}
}

// TestClientEndpointAuthStyle is a compile-time style check that a derived
// client carries a usable auth style rather than the zero value.
func TestDerivedClientCarriesAuthStyle(t *testing.T) {
	t.Parallel()

	gitea, err := Derive(KindGitea, "https://git.example.com")
	require.NoError(t, err)
	assert.Equal(t, oauth2.AuthStyleInParams, gitea.Endpoint.AuthStyle)
	assert.True(t, gitea.PKCE)
	assert.False(t, gitea.DeviceFlow)
}
