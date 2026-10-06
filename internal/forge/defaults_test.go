// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

// TestBuiltinClientsMatchTheDeclaredCount checks the table and constant agree.
func TestBuiltinClientsMatchTheDeclaredCount(t *testing.T) {
	t.Parallel()

	clients, err := builtinClients()
	require.NoError(t, err)
	assert.Len(t, clients, builtinHostCount)

	seen := map[string]bool{}
	for _, client := range clients {
		assert.False(t, seen[client.Host], "host %q is listed twice", client.Host)
		seen[client.Host] = true
	}
}

// TestEveryGitLabHostIsRecognizedAsGitLab covers the hosts without a prefix.
//
// Several public GitLab hosts, such as salsa.debian.org, carry no gitlab.
// prefix, so the table is the only thing that classifies them.
func TestEveryGitLabHostIsRecognizedAsGitLab(t *testing.T) {
	t.Parallel()

	table := New()

	for _, host := range gitlabHosts() {
		client, ok := table.Lookup(host)
		require.True(t, ok, "host %q", host)

		assert.Equal(t, KindGitLab, client.Kind, "host %q", host)
		assert.Equal(t, "https://"+host+"/oauth/token", client.Endpoint.TokenURL, "host %q", host)
		assert.Equal(t, "https://"+host+"/oauth/authorize_device", client.Endpoint.DeviceAuthURL, "host %q", host)
		assert.True(t, client.DeviceFlow, "host %q", host)
	}
}

// TestGitLabEndpointUsesTheLibraryEndpointForGitLabCom covers the shortcut.
func TestGitLabEndpointUsesTheLibraryEndpointForGitLabCom(t *testing.T) {
	t.Parallel()

	endpoint, err := gitLabEndpoint("gitlab.com")
	require.NoError(t, err)
	assert.Equal(t, endpoints.GitLab, endpoint)

	_, err = gitLabEndpoint("")
	require.ErrorIs(t, err, errHostEmpty)
}

// TestReplaceHostKeepsPathAndAuthStyle checks only the host changes.
func TestReplaceHostKeepsPathAndAuthStyle(t *testing.T) {
	t.Parallel()

	source := oauth2.Endpoint{
		AuthURL:   "https://gitlab.com/oauth/authorize",
		TokenURL:  "https://gitlab.com/oauth/token",
		AuthStyle: oauth2.AuthStyleInHeader,
	}

	got, err := replaceHost(source, "gitlab.example.com:8443")
	require.NoError(t, err)

	assert.Equal(t, "https://gitlab.example.com:8443/oauth/authorize", got.AuthURL)
	assert.Equal(t, "https://gitlab.example.com:8443/oauth/token", got.TokenURL)
	assert.Empty(t, got.DeviceAuthURL, "an empty device URL stays empty")
	assert.Equal(t, oauth2.AuthStyleInHeader, got.AuthStyle)
}

// TestReplaceHostRejectsBadInput covers each URL that can fail.
func TestReplaceHostRejectsBadInput(t *testing.T) {
	t.Parallel()

	good := "https://gitlab.com/x"

	tests := []struct {
		want     error
		name     string
		host     string
		endpoint oauth2.Endpoint
	}{
		{name: "empty host", endpoint: oauth2.Endpoint{AuthURL: good}, host: "", want: errHostEmpty},
		{name: "relative auth URL", endpoint: oauth2.Endpoint{AuthURL: "/x", TokenURL: good}, host: "h", want: errMalformedURL},
		{name: "relative token URL", endpoint: oauth2.Endpoint{AuthURL: good, TokenURL: "/x"}, host: "h", want: errMalformedURL},
		{
			name:     "relative device URL",
			endpoint: oauth2.Endpoint{AuthURL: good, TokenURL: good, DeviceAuthURL: "/x"},
			host:     "h",
			want:     errMalformedURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := replaceHost(tt.endpoint, tt.host)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

// TestReplaceHostInURLReportsAParseFailure covers an unparseable URL.
func TestReplaceHostInURLReportsAParseFailure(t *testing.T) {
	t.Parallel()

	_, err := replaceHostInURL("https://gitlab.com/%zz", "h")
	require.Error(t, err)

	var parseErr *url.Error
	require.ErrorAs(t, err, &parseErr)
}

// TestServerRegisteredClientsUseTheGiteaLayout checks gitea.com and codeberg.org.
func TestServerRegisteredClientsUseTheGiteaLayout(t *testing.T) {
	t.Parallel()

	clients, err := giteaClients()
	require.NoError(t, err)
	require.Len(t, clients, 2)

	for _, client := range clients {
		assert.Equal(t, universalGiteaClientID, client.ClientID, "host %q", client.Host)
		assert.Equal(t, "https://"+client.Host+"/login/oauth/access_token", client.Endpoint.TokenURL)
		assert.Equal(t, oauth2.AuthStyleInParams, client.Endpoint.AuthStyle)
		assert.False(t, client.DeviceFlow, "host %q", client.Host)
	}
}
