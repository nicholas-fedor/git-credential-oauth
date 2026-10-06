// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/oauth2"
)

// metadataServer serves body at the well-known path for host.
//
// The server is TLS because discovery only ever fetches https. A test that
// needs to see a cleartext attempt has to assert on the endpoint check instead.
func metadataServer(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, metadataPath, r.URL.Path, "metadata is fetched from the well-known path")

		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))

	t.Cleanup(server.Close)

	return server
}

// hostOf returns the hostname a test server is reachable at.
func hostOf(server *httptest.Server) string {
	return server.Listener.Addr().String()
}

// TestDiscoverReadsAPublishedDocument covers the ordinary case.
//
// The document is generated from the request host so the issuer matches, which
// is what RFC 8414 section 3.3 requires of a real one.
func TestDiscoverReadsAPublishedDocument(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, metadataPath, r.URL.Path, "metadata is fetched from the well-known path")

		issuer := "https://" + r.Host
		fmt.Fprintf(w, `{
			"issuer": %q,
			"authorization_endpoint": %q,
			"token_endpoint": %q,
			"device_authorization_endpoint": %q,
			"code_challenge_methods_supported": ["S256"]
		}`,
			issuer,
			issuer+"/oauth/authorize",
			issuer+"/oauth/token",
			issuer+"/oauth/device",
		)
	}))
	t.Cleanup(server.Close)

	host := hostOf(server)

	meta, found, err := Discover(t.Context(), host, server.Client())
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, "https://"+host+"/oauth/authorize", meta.AuthorizationEndpoint)
	assert.Equal(t, "https://"+host+"/oauth/token", meta.TokenEndpoint)
	assert.Equal(t, "https://"+host+"/oauth/device", meta.DeviceAuthorizationEndpoint)
	assert.True(t, meta.SupportsDeviceFlow(), "device endpoint advertised")
	assert.True(t, meta.SupportsPKCES256(), "S256 advertised")
}

// TestDiscoverRejectsADuplicatedEndpoint is why this uses encoding/json/v2.
//
// A document naming token_endpoint twice is ambiguous, and which value a parser
// keeps is not something a client should depend on. v2 rejects the document
// rather than silently keeping one of them, which leaves the derived endpoints
// in place rather than sending a token to whichever value won.
func TestDiscoverRejectsADuplicatedEndpoint(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		fmt.Fprintf(w,
			`{"issuer": "https://%[1]s",`+
				`"token_endpoint": "https://%[1]s/good",`+
				`"token_endpoint": "https://%[1]s/decoy"}`,
			host)
	}))
	t.Cleanup(server.Close)

	_, found, err := Discover(t.Context(), hostOf(server), server.Client())

	require.ErrorIs(t, err, errMetadataBody)
	assert.False(t, found)
}

// TestDiscoverTreatsAbsentMetadataAsNotFound is the common case.
//
// Most forges publish no document, so this must not be an error or the caller
// would refuse to fall back to the built-in client.
func TestDiscoverTreatsAbsentMetadataAsNotFound(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusNotFound, http.StatusMethodNotAllowed} {
		server := metadataServer(t, "", status)

		_, found, err := Discover(t.Context(), hostOf(server), server.Client())

		require.ErrorIs(t, err, errMetadataAbsent)
		assert.False(t, found)
	}
}

// TestDiscoverRejectsAnotherHostsIssuer is the attack this feature prevents.
//
// A document naming a different issuer means the host served someone else's
// endpoints, which is exactly the misconfiguration RFC 9700 section 2.6 warns
// about. Following it would send the token to the wrong party.
func TestDiscoverRejectsAnotherHostsIssuer(t *testing.T) {
	t.Parallel()

	const document = `{"issuer": "https://evil.example.com",` +
		`"token_endpoint": "https://evil.example.com/t"}`

	server := metadataServer(t, document, http.StatusOK)

	_, found, err := Discover(t.Context(), hostOf(server), server.Client())

	require.ErrorIs(t, err, errMetadataIssuer)
	assert.False(t, found)
}

// TestDiscoverRejectsAnEmptyIssuer covers a document with no issuer.
func TestDiscoverRejectsAnEmptyIssuer(t *testing.T) {
	t.Parallel()

	server := metadataServer(t, `{"token_endpoint": "https://x.example.com/t"}`, http.StatusOK)

	_, found, err := Discover(t.Context(), hostOf(server), server.Client())

	require.ErrorIs(t, err, errMetadataIssuer)
	assert.False(t, found)
}

// TestDiscoverRejectsAMalformedDocument covers a body that is not JSON.
func TestDiscoverRejectsAMalformedDocument(t *testing.T) {
	t.Parallel()

	server := metadataServer(t, "not json at all", http.StatusOK)

	_, found, err := Discover(t.Context(), hostOf(server), server.Client())

	require.ErrorIs(t, err, errMetadataBody)
	assert.False(t, found)
}

// TestDiscoverSurfacesAServerFailure covers an unexpected status.
func TestDiscoverSurfacesAServerFailure(t *testing.T) {
	t.Parallel()

	server := metadataServer(t, "", http.StatusInternalServerError)

	_, _, err := Discover(t.Context(), hostOf(server), server.Client())

	require.ErrorIs(t, err, errMetadataStatus)
}

// TestDiscoverHonorsCancellation checks a canceled context stops the fetch.
func TestDiscoverHonorsCancellation(t *testing.T) {
	t.Parallel()

	server := metadataServer(t, "{}", http.StatusOK)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := Discover(ctx, hostOf(server), server.Client())

	require.Error(t, err)
}

// TestSupportsPKCES256DefaultsToTrue covers the missing list.
//
// RFC 6749 does not make code_challenge_methods_supported mandatory, so a
// document that omits it is not refusing S256.
func TestSupportsPKCES256DefaultsToTrue(t *testing.T) {
	t.Parallel()

	assert.True(t, Metadata{}.SupportsPKCES256())
	assert.True(t, Metadata{
		CodeChallengeMethodsSupported: []string{},
	}.SupportsPKCES256())
}

// TestSupportsPKCES256RejectsPlainOnly covers a provider that cannot take S256.
func TestSupportsPKCES256RejectsPlainOnly(t *testing.T) {
	t.Parallel()

	meta := Metadata{CodeChallengeMethodsSupported: []string{"plain"}}

	assert.False(t, meta.SupportsPKCES256(), "this client only sends S256")
	assert.True(t, Metadata{
		CodeChallengeMethodsSupported: []string{"plain", "S256"},
	}.SupportsPKCES256())
}

// TestSupportsDeviceFlowOnlyWhenAdvertised covers the device endpoint check.
func TestSupportsDeviceFlowOnlyWhenAdvertised(t *testing.T) {
	t.Parallel()

	assert.False(t, Metadata{}.SupportsDeviceFlow())
	assert.True(t, Metadata{
		DeviceAuthorizationEndpoint: "https://example.com/device",
	}.SupportsDeviceFlow())
}

// TestFromMetadataUsesPublishedEndpoints is the point of the feature.
func TestFromMetadataUsesPublishedEndpoints(t *testing.T) {
	t.Parallel()

	meta := Metadata{
		Issuer:                        "https://git.example.com",
		AuthorizationEndpoint:         "https://sso.example.com/authorize",
		TokenEndpoint:                 "https://sso.example.com/token",
		DeviceAuthorizationEndpoint:   "https://sso.example.com/device",
		CodeChallengeMethodsSupported: []string{"S256"},
	}

	client, found, err := FromMetadata(KindGitLab, "git.example.com", meta)
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, "https://sso.example.com/authorize", client.Endpoint.AuthURL)
	assert.Equal(t, "https://sso.example.com/token", client.Endpoint.TokenURL)
	assert.Equal(t, "https://sso.example.com/device", client.Endpoint.DeviceAuthURL)
	assert.True(t, client.PKCE)
	assert.Equal(t, oauth2.AuthStyleAutoDetect, client.Endpoint.AuthStyle)
}

// TestFromMetadataDisablesPKCEWhenUnsupported covers the capability check.
//
// This client always sends an S256 challenge. A provider that lists only plain
// would reject it, so following such a document is worse than ignoring it.
func TestFromMetadataDisablesPKCEWhenUnsupported(t *testing.T) {
	t.Parallel()

	meta := Metadata{
		Issuer:                        "https://git.example.com",
		AuthorizationEndpoint:         "https://git.example.com/authorize",
		TokenEndpoint:                 "https://git.example.com/token",
		CodeChallengeMethodsSupported: []string{"plain"},
	}

	client, found, err := FromMetadata(KindGitea, "git.example.com", meta)
	require.NoError(t, err)
	require.True(t, found)
	assert.False(t, client.PKCE, "a provider that cannot take S256")
}

// TestFromMetadataNeedsBothEndpoints covers an incomplete document.
func TestFromMetadataNeedsBothEndpoints(t *testing.T) {
	t.Parallel()

	partials := []Metadata{
		{TokenEndpoint: "https://git.example.com/token"},
		{AuthorizationEndpoint: "https://git.example.com/authorize"},
		{},
	}

	for _, meta := range partials {
		_, found, err := FromMetadata(KindGitLab, "git.example.com", meta)
		require.NoError(t, err)
		assert.False(t, found, "an unusable document is skipped, not fatal")
	}
}

// TestFromMetadataRejectsCleartextEndpoints is the security boundary.
//
// The metadata document decides where the authorization code and the token are
// sent. A cleartext endpoint would put both on the wire in the open.
func TestFromMetadataRejectsCleartextEndpoints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		meta Metadata
	}{
		{
			name: "cleartext authorization",
			meta: Metadata{
				AuthorizationEndpoint: "http://git.example.com/authorize",
				TokenEndpoint:         "https://git.example.com/token",
			},
		},
		{
			name: "cleartext token",
			meta: Metadata{
				AuthorizationEndpoint: "https://git.example.com/authorize",
				TokenEndpoint:         "http://git.example.com/token",
			},
		},
		{
			name: "cleartext device",
			meta: Metadata{
				AuthorizationEndpoint:       "https://git.example.com/authorize",
				TokenEndpoint:               "https://git.example.com/token",
				DeviceAuthorizationEndpoint: "http://git.example.com/device",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, found, err := FromMetadata(KindGitLab, "git.example.com", tt.meta)

			require.ErrorIs(t, err, errInsecureGitURL)
			assert.False(t, found)
		})
	}
}

// TestFromMetadataKeepsTheFamilyClientID checks the family still decides
// whether an operator has to register anything.
func TestFromMetadataKeepsTheFamilyClientID(t *testing.T) {
	t.Parallel()

	meta := Metadata{
		AuthorizationEndpoint: "https://git.example.com/authorize",
		TokenEndpoint:         "https://git.example.com/token",
	}

	gitea, _, err := FromMetadata(KindGitea, "git.example.com", meta)
	require.NoError(t, err)
	assert.Equal(t, universalGiteaClientID, gitea.ClientID)

	gitlab, _, err := FromMetadata(KindGitLab, "git.example.com", meta)
	require.NoError(t, err)
	assert.Empty(t, gitlab.ClientID, "GitLab has no published application")
}

// TestIssuerHostPathUsesHTTPS checks the fetch is never cleartext.
func TestIssuerHostPathUsesHTTPS(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		"https://git.example.com"+metadataPath,
		issuerHostPath("git.example.com"))
	assert.Equal(t,
		"https://git.example.com:8443"+metadataPath,
		issuerHostPath("git.example.com:8443"))
}
