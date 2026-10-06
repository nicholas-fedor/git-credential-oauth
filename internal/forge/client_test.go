// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/oauth2"
)

// TestParseGitURLRejectsCleartext covers the transport guard.
//
// A token issued for an http remote would be sent in the clear, so the remote
// root itself is refused before any endpoint is derived from it.
func TestParseGitURLRejectsCleartext(t *testing.T) {
	t.Parallel()

	for _, gitURL := range []string{"http://git.example.com", "HTTP://git.example.com", "ftp://git.example.com"} {
		_, err := parseGitURL(gitURL)
		require.ErrorIs(t, err, errInsecureGitURL, "git URL %q", gitURL)
	}

	parsed, err := parseGitURL("https://git.example.com:8443")
	require.NoError(t, err)
	assert.Equal(t, "git.example.com:8443", parsed.Host)
}

// TestParseGitURLRejectsIncompleteURLs covers the shape checks.
func TestParseGitURLRejectsIncompleteURLs(t *testing.T) {
	t.Parallel()

	_, err := parseGitURL("://bad")
	require.ErrorIs(t, err, errParseGitURL)

	for _, gitURL := range []string{"", "git.example.com", "https://", "/path"} {
		_, err = parseGitURL(gitURL)
		require.ErrorIs(t, err, errMalformedGitURL, "git URL %q", gitURL)
	}
}

// TestHostFromGitURL keeps the port and drops everything else.
func TestHostFromGitURL(t *testing.T) {
	t.Parallel()

	host, err := hostFromGitURL("https://git.example.com:8443/group/repo.git")
	require.NoError(t, err)
	assert.Equal(t, "git.example.com:8443", host)

	_, err = hostFromGitURL("http://git.example.com")
	require.ErrorIs(t, err, errInsecureGitURL)
}

// TestAppendPathJoinsOnce checks a trailing slash on the root is not doubled.
func TestAppendPathJoinsOnce(t *testing.T) {
	t.Parallel()

	for _, root := range []string{"https://git.example.com", "https://git.example.com/", "https://git.example.com//"} {
		got, err := appendPath(root, "/oauth/token")
		require.NoError(t, err)
		assert.Equal(t, "https://git.example.com/oauth/token", got, "root %q", root)
	}

	_, err := appendPath("http://git.example.com", "/oauth/token")
	require.ErrorIs(t, err, errInsecureGitURL)
}

// TestEndpointForLeavesTheDeviceURLEmptyWithoutAPath covers families without
// the device grant.
func TestEndpointForLeavesTheDeviceURLEmptyWithoutAPath(t *testing.T) {
	t.Parallel()

	endpoint, err := endpointFor("https://git.example.com", "/a", "/t", "", oauth2.AuthStyleInParams)
	require.NoError(t, err)

	assert.Equal(t, "https://git.example.com/a", endpoint.AuthURL)
	assert.Equal(t, "https://git.example.com/t", endpoint.TokenURL)
	assert.Empty(t, endpoint.DeviceAuthURL)
	assert.Equal(t, oauth2.AuthStyleInParams, endpoint.AuthStyle)

	_, err = endpointFor("http://git.example.com", "/a", "/t", "/d", oauth2.AuthStyleInParams)
	require.ErrorIs(t, err, errInsecureGitURL)
}

// TestDeviceAuthURL covers the optional device path.
func TestDeviceAuthURL(t *testing.T) {
	t.Parallel()

	got, err := deviceAuthURL("https://git.example.com", "")
	require.NoError(t, err)
	assert.Empty(t, got)

	got, err = deviceAuthURL("https://git.example.com", "/device")
	require.NoError(t, err)
	assert.Equal(t, "https://git.example.com/device", got)

	_, err = deviceAuthURL("http://git.example.com", "/device")
	require.ErrorIs(t, err, errInsecureGitURL)
}

// TestHTTPSEndpointRejectsEveryCleartextURL covers each published endpoint.
func TestHTTPSEndpointRejectsEveryCleartextURL(t *testing.T) {
	t.Parallel()

	base := Metadata{
		AuthorizationEndpoint:       "https://git.example.com/authorize",
		TokenEndpoint:               "https://git.example.com/token",
		DeviceAuthorizationEndpoint: "https://git.example.com/device",
	}

	endpoint, err := httpsEndpoint(KindGitLab, base)
	require.NoError(t, err)
	assert.Equal(t, base.DeviceAuthorizationEndpoint, endpoint.DeviceAuthURL)

	unparseable := base
	unparseable.TokenEndpoint = "https://git.example.com/%zz"

	_, err = httpsEndpoint(KindGitLab, unparseable)
	require.ErrorIs(t, err, errInsecureGitURL)

	relative := base
	relative.AuthorizationEndpoint = "/authorize"

	_, err = httpsEndpoint(KindGitLab, relative)
	require.ErrorIs(t, err, errInsecureGitURL, "a relative endpoint has no https scheme")
}

// TestAuthStyleFor sends Gitea-layout credentials in the request body.
func TestAuthStyleFor(t *testing.T) {
	t.Parallel()

	for _, kind := range []Kind{KindGitea, KindForgejo} {
		assert.Equal(t, oauth2.AuthStyleInParams, authStyleFor(kind), "kind %s", kind)
	}

	for _, kind := range []Kind{KindGitHub, KindGitHubEnterprise, KindGitLab, KindBitbucket, KindUnknown} {
		assert.Equal(t, oauth2.AuthStyleAutoDetect, authStyleFor(kind), "kind %s", kind)
	}
}

// TestPublicClientIDForOnlyGiteaFamilies checks no other family gains an ID.
//
// The shared ID exists only on servers that register it. Sending it anywhere
// else would present an application the forge has never heard of.
func TestPublicClientIDForOnlyGiteaFamilies(t *testing.T) {
	t.Parallel()

	for kind := KindUnknown; int(kind) < kindCount; kind++ {
		want := ""
		if kind == KindGitea || kind == KindForgejo {
			want = universalGiteaClientID
		}

		assert.Equal(t, want, publicClientIDFor(kind), "kind %s", kind)
	}
}

// TestMetadataScopesPerFamily pins the scopes requested from published
// endpoints.
func TestMetadataScopesPerFamily(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"read:repository", "write:repository"}, metadataScopes(KindForgejo))
	assert.Equal(t, []string{"repo", "gist", "workflow"}, metadataScopes(KindGitHubEnterprise))
	assert.Equal(t, []string{"read_repository", "write_repository"}, metadataScopes(KindGitLab))
	assert.Nil(t, metadataScopes(KindGitea))
	assert.Nil(t, metadataScopes(KindUnknown))
}

// TestCloneClientIsolatesScopes checks a copy cannot rewrite its source.
func TestCloneClientIsolatesScopes(t *testing.T) {
	t.Parallel()

	original := Client{Host: "git.example.com", Scopes: []string{"a", "b"}}
	copied := cloneClient(original)
	copied.Scopes[0] = "changed"

	assert.Equal(t, "a", original.Scopes[0])
}

// TestNewClientShipsNoSecret checks the constructor cannot carry one.
//
// A secret may only come from the operator's Git config, so every client this
// package builds starts without one, with PKCE on.
func TestNewClientShipsNoSecret(t *testing.T) {
	t.Parallel()

	scopes := []string{"repo"}
	client := newClient("git.example.com", KindGitHub, "id", scopes, oauth2.Endpoint{}, true)
	scopes[0] = "changed"

	assert.Empty(t, client.ClientSecret)
	assert.True(t, client.PKCE)
	assert.True(t, client.DeviceFlow)
	assert.Equal(t, []string{"repo"}, client.Scopes, "the scope slice is copied")
}

// TestFromMetadataReportsDeviceSupportFromTheDocument covers the device flag.
func TestFromMetadataReportsDeviceSupportFromTheDocument(t *testing.T) {
	t.Parallel()

	meta := Metadata{
		Issuer:                "https://git.example.com",
		AuthorizationEndpoint: "https://git.example.com/authorize",
		TokenEndpoint:         "https://git.example.com/token",
	}

	client, found, err := FromMetadata(KindGitea, "git.example.com", meta)
	require.NoError(t, err)
	require.True(t, found)
	assert.False(t, client.DeviceFlow)
	assert.Equal(t, oauth2.AuthStyleInParams, client.Endpoint.AuthStyle)

	meta.DeviceAuthorizationEndpoint = "https://git.example.com/device"

	client, _, err = FromMetadata(KindGitea, "git.example.com", meta)
	require.NoError(t, err)
	assert.True(t, client.DeviceFlow)
}
