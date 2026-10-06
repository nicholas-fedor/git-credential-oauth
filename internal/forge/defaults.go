// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"fmt"
	"net/url"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

const (
	// builtinHostCount is the number of hosts in the built-in registry.
	builtinHostCount = 15

	// hostGitHub is the public GitHub host.
	hostGitHub = "github.com"
	// hostGistGitHub is the GitHub gist host.
	hostGistGitHub = "gist.github.com"
	// hostGitLab is the public GitLab host.
	hostGitLab = "gitlab.com"
	// hostGitLabFreedesktop is the freedesktop.org GitLab host.
	hostGitLabFreedesktop = "gitlab.freedesktop.org"
	// hostGitLabGNOME is the GNOME GitLab host.
	hostGitLabGNOME = "gitlab.gnome.org"
	// hostVideoLAN is the VideoLAN GitLab host.
	hostVideoLAN = "code.videolan.org"
	// hostSalsa is the Debian Salsa host.
	hostSalsa = "salsa.debian.org"
	// hostGitLabHaskell is the Haskell GitLab host.
	hostGitLabHaskell = "gitlab.haskell.org"
	// hostGitLabAlpine is the Alpine GitLab host.
	hostGitLabAlpine = "gitlab.alpinelinux.org"
	// hostHelmholtz is the Helmholtz GitLab host.
	hostHelmholtz = "codebase.helmholtz.cloud"
	// hostKDE is the KDE GitLab host.
	hostKDE = "invent.kde.org"
	// hostGitea is the public Gitea host.
	hostGitea = "gitea.com"
	// hostCodeberg is the public Forgejo host.
	hostCodeberg = "codeberg.org"
	// hostBitbucket is the public Bitbucket host.
	hostBitbucket = "bitbucket.org"
	// hostAndroidGoogleSource is the Android Google Source host.
	hostAndroidGoogleSource = "android.googlesource.com"

	// scopeRepo is the GitHub repository scope.
	scopeRepo = "repo"
	// scopeGist is the GitHub gist scope.
	scopeGist = "gist"
	// scopeWorkflow is the GitHub workflow scope.
	scopeWorkflow = "workflow"
	// scopeReadRepository is the GitLab read-repository scope.
	scopeReadRepository = "read_repository"
	// scopeWriteRepository is the GitLab write-repository scope.
	scopeWriteRepository = "write_repository"
	// scopeReadRepoColon is the Forgejo read-repository scope.
	scopeReadRepoColon = "read:repository"
	// scopeWriteRepoColon is the Forgejo write-repository scope.
	scopeWriteRepoColon = "write:repository"
	// scopeBitbucketRepo is the Bitbucket repository scope.
	scopeBitbucketRepo = "repository"
	// scopeBitbucketWrite is the Bitbucket repository-write scope.
	scopeBitbucketWrite = "repository:write"
	// scopeGerrit is the Google Source Gerrit scope.
	scopeGerrit = "https://www.googleapis.com/auth/gerritcodereview"
)

// builtinClients returns every public host, including gist.github.com.
//
// An entry records what this program knows about a host: its family, its
// endpoints, and the scopes Git needs. It carries no application of this
// project's or anyone else's. The operator registers their own on each forge
// and supplies it through credential.oauthClientId, so the consent screen names
// an application they control. The exceptions are gitea.com and codeberg.org,
// whose servers register an application for this helper themselves.
//
// Returns:
//   - []Client: built-in clients.
//   - error: a public endpoint URL could not be rewritten.
func builtinClients() ([]Client, error) {
	clients := make([]Client, 0, builtinHostCount)

	clients = append(clients, githubClients()...)

	gitlab, err := gitLabClients()
	if err != nil {
		return nil, err
	}

	clients = append(clients, gitlab...)

	gitea, err := giteaClients()
	if err != nil {
		return nil, err
	}

	clients = append(clients, gitea...)
	clients = append(clients, bitbucketClient(), googleClient())

	return clients, nil
}

// bitbucketClient returns the public Bitbucket client.
//
// Returns:
//   - Client: bitbucket.org client using endpoints.Bitbucket, awaiting operator
//     keys.
func bitbucketClient() Client {
	return newClient(
		hostBitbucket,
		KindBitbucket,
		"",
		bitbucketScopes(),
		endpoints.Bitbucket,
		false,
	)
}

// bitbucketScopes returns a fresh Bitbucket scope slice.
//
// Returns:
//   - []string: repository and repository:write.
func bitbucketScopes() []string {
	return []string{scopeBitbucketRepo, scopeBitbucketWrite}
}

// forgejoScopes returns a fresh Forgejo scope slice.
//
// Forgejo uses colon scopes, not GitLab's read_repository form.
//
// Returns:
//   - []string: read:repository and write:repository.
func forgejoScopes() []string {
	return []string{scopeReadRepoColon, scopeWriteRepoColon}
}

// gitLabClients returns public and vendor GitLab clients.
//
// Returns:
//   - []Client: one client per GitLab host in the built-in table.
//   - error: a vendor endpoint URL could not be rewritten.
func gitLabClients() ([]Client, error) {
	hosts := gitlabHosts()
	clients := make([]Client, 0, len(hosts))

	for _, host := range hosts {
		client, err := newGitLabClient(host)
		if err != nil {
			return nil, err
		}

		clients = append(clients, client)
	}

	return clients, nil
}

// gitLabEndpoint returns endpoints.GitLab, or a host-replaced copy.
//
// Replacement runs from New, never from init, and returns an error on a
// malformed URL instead of exiting.
//
// Parameters:
//   - host: GitLab hostname. gitlab.com uses the endpoint unchanged.
//
// Returns:
//   - oauth2.Endpoint: GitLab authorize, token, and device URLs.
//   - error: host is empty or an endpoint URL is malformed.
func gitLabEndpoint(host string) (oauth2.Endpoint, error) {
	if host == hostGitLab {
		return endpoints.GitLab, nil
	}

	endpoint, err := replaceHost(endpoints.GitLab, host)
	if err != nil {
		return oauth2.Endpoint{}, err
	}

	return endpoint, nil
}

// gitLabScopes returns a fresh GitLab scope slice.
//
// Returns:
//   - []string: read_repository and write_repository.
func gitLabScopes() []string {
	return []string{scopeReadRepository, scopeWriteRepository}
}

// giteaClients returns gitea.com and codeberg.org.
//
// Both are registry entries so their apex names are not mistaken for
// self-hosted instances. gitea.com keeps an empty scope list. codeberg.org uses
// Forgejo colon scopes. Both use the application their own server registers.
//
// Returns:
//   - []Client: public Gitea and Forgejo clients.
//   - error: an endpoint URL could not be built.
func giteaClients() ([]Client, error) {
	gitea, err := newGiteaComClient()
	if err != nil {
		return nil, err
	}

	codeberg, err := newCodebergClient()
	if err != nil {
		return nil, err
	}

	return []Client{gitea, codeberg}, nil
}

// githubClients returns github.com and gist.github.com.
//
// Returns:
//   - []Client: public GitHub clients, awaiting operator keys.
func githubClients() []Client {
	return []Client{
		newGitHubClient(hostGitHub),
		newGitHubClient(hostGistGitHub),
	}
}

// githubScopes returns a fresh GitHub scope slice.
//
// Returns:
//   - []string: repo, gist, and workflow.
func githubScopes() []string {
	return []string{scopeRepo, scopeGist, scopeWorkflow}
}

// gitlabHosts lists the public GitLab hosts recognized by name.
//
// Several of them carry no gitlab. prefix, so without an entry here they would
// be classified only when the server sends a GitLab realm.
//
// Returns:
//   - []string: GitLab host names.
func gitlabHosts() []string {
	return []string{
		hostGitLab,
		hostGitLabFreedesktop,
		hostGitLabGNOME,
		hostVideoLAN,
		hostSalsa,
		hostGitLabHaskell,
		hostGitLabAlpine,
		hostHelmholtz,
		hostKDE,
	}
}

// googleClient returns the android.googlesource.com client.
//
// Returns:
//   - Client: browser-flow Google Source client, awaiting operator keys.
func googleClient() Client {
	return newClient(
		hostAndroidGoogleSource,
		KindGoogleSource,
		"",
		googleScopes(),
		endpoints.Google,
		false,
	)
}

// googleScopes returns a fresh Google Source scope slice.
//
// Returns:
//   - []string: the Gerrit code-review scope.
func googleScopes() []string {
	return []string{scopeGerrit}
}

// newCodebergClient returns the Forgejo client for codeberg.org.
//
// Returns:
//   - Client: codeberg.org client with colon scopes and the application
//     Forgejo registers itself.
//   - error: the endpoint URL could not be built.
func newCodebergClient() (Client, error) {
	endpoint, err := giteaStyleEndpoint("https://" + hostCodeberg)
	if err != nil {
		return Client{}, err
	}

	return newClient(
		hostCodeberg,
		KindForgejo,
		universalGiteaClientID,
		forgejoScopes(),
		endpoint,
		false,
	), nil
}

// newGitHubClient returns a public GitHub client for host.
//
// The client ID and secret are deliberately empty. An operator registers their
// own application and sets credential.oauthClientId, and
// credential.oauthClientSecret for the browser grant, which GitHub requires at
// the token endpoint. Device flow needs no secret.
//
// Parameters:
//   - host: github.com or gist.github.com.
//
// Returns:
//   - Client: GitHub client using endpoints.GitHub, awaiting operator keys.
func newGitHubClient(host string) Client {
	return newClient(
		host,
		KindGitHub,
		"",
		githubScopes(),
		endpoints.GitHub,
		true,
	)
}

// newGitLabClient returns one GitLab table entry.
//
// Parameters:
//   - host: GitLab hostname.
//
// Returns:
//   - Client: GitLab client for host, awaiting an operator client ID.
//   - error: the endpoint URL could not be rewritten.
func newGitLabClient(host string) (Client, error) {
	endpoint, err := gitLabEndpoint(host)
	if err != nil {
		return Client{}, err
	}

	return newClient(
		host,
		KindGitLab,
		"",
		gitLabScopes(),
		endpoint,
		true,
	), nil
}

// newGiteaComClient returns the public Gitea client.
//
// Scopes stay nil, matching the legacy table.
//
// Returns:
//   - Client: gitea.com client with the application Gitea registers itself.
//   - error: the endpoint URL could not be built.
func newGiteaComClient() (Client, error) {
	endpoint, err := giteaStyleEndpoint("https://" + hostGitea)
	if err != nil {
		return Client{}, err
	}

	return newClient(
		hostGitea,
		KindGitea,
		universalGiteaClientID,
		nil,
		endpoint,
		false,
	), nil
}

// replaceHost returns a copy of endpoint with host replaced in each URL.
//
// AuthStyle is preserved. An empty device URL stays empty.
//
// Parameters:
//   - endpoint: source endpoint, usually endpoints.GitLab.
//   - host: replacement hostname.
//
// Returns:
//   - oauth2.Endpoint: copy with host replaced.
//   - error: host is empty or a URL is malformed.
func replaceHost(
	endpoint oauth2.Endpoint,
	host string,
) (oauth2.Endpoint, error) {
	if host == "" {
		return oauth2.Endpoint{}, fmt.Errorf("replace host: %w", errHostEmpty)
	}

	authURL, err := replaceHostInURL(endpoint.AuthURL, host)
	if err != nil {
		return oauth2.Endpoint{}, fmt.Errorf("replace auth URL host: %w", err)
	}

	tokenURL, err := replaceHostInURL(endpoint.TokenURL, host)
	if err != nil {
		return oauth2.Endpoint{}, fmt.Errorf("replace token URL host: %w", err)
	}

	deviceURL, err := replaceHostInURL(endpoint.DeviceAuthURL, host)
	if err != nil {
		return oauth2.Endpoint{}, fmt.Errorf("replace device URL host: %w", err)
	}

	return oauth2.Endpoint{
		AuthURL:       authURL,
		DeviceAuthURL: deviceURL,
		TokenURL:      tokenURL,
		AuthStyle:     endpoint.AuthStyle,
	}, nil
}

// replaceHostInURL replaces the host in originalURL.
//
// An empty URL is preserved. A malformed URL returns an error.
//
// Parameters:
//   - originalURL: absolute URL whose host is replaced. Empty is preserved.
//   - host: replacement hostname.
//
// Returns:
//   - string: URL with the new host, or empty when originalURL is empty.
//   - error: originalURL cannot be parsed.
func replaceHostInURL(originalURL, host string) (string, error) {
	if originalURL == "" {
		return "", nil
	}

	parsed, err := url.Parse(originalURL)
	if err != nil {
		return "", fmt.Errorf("parse URL %q: %w", originalURL, err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("%w: %q", errMalformedURL, originalURL)
	}

	parsed.Host = host

	return parsed.String(), nil
}
