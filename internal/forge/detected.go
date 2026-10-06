// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

// Detected is the classification of one Git host.
//
// The forge side of a remote only: the transport scheme and the host are
// already known by the caller, and Scheme would only repeat the transport that
// the URL already carries.
type Detected struct {
	// Host is the Git hostname, compared as given.
	Host string

	// Realm is the WWW-Authenticate realm, empty when the host sent none.
	Realm string

	// Kind is the forge family.
	Kind Kind
}

const (
	// usernameTokenAuth is the Bitbucket credential username.
	usernameTokenAuth = "x-token-auth"
	// usernameCredentialOAuth2 is the username emitted for every forge except
	// Bitbucket. It is a placeholder, not an account name.
	usernameCredentialOAuth2 = "oauth2"
	// loginParamGitHub is the GitHub authorization-URL login parameter.
	loginParamGitHub = "login"
	// loginParamGoogle is the Google authorization-URL login parameter.
	loginParamGoogle = "login_hint"

	// loopbackRedirect is the redirect URI an operator registers. It names no
	// port because the helper binds a free one for each request, and RFC 8252
	// section 7.3 has the server accept any port on a loopback address.
	loopbackRedirect = "http://127.0.0.1"

	// keyOAuthClientID is the config key suffix for the client ID.
	keyOAuthClientID = "oauthClientId"
	// keyOAuthClientSecret is the config key suffix for the client secret.
	keyOAuthClientSecret = "oauthClientSecret"
	// keyOAuthAuthURL is the config key suffix for the authorization endpoint.
	keyOAuthAuthURL = "oauthAuthURL"
	// keyOAuthTokenURL is the config key suffix for the token endpoint.
	keyOAuthTokenURL = "oauthTokenURL"

	// pathGitHubNewApplication is the GitHub application form.
	pathGitHubNewApplication = "/settings/applications/new"
	// pathGitLabApplications is the GitLab user application form.
	pathGitLabApplications = "/-/user_settings/applications"

	// registerAppSentence is the setup hint for kinds without a known form.
	registerAppSentence = "Register an OAuth application on this host with " +
		"redirect URI " + loopbackRedirect + "."
	// bitbucketSetupSentence is the setup hint for Bitbucket. The consumer form
	// sits under a workspace the operator has to pick, so it has no fixed URL.
	bitbucketSetupSentence = "Add an OAuth consumer under your Bitbucket " +
		"workspace settings with callback URL " + loopbackRedirect +
		" and repository read and write permissions."
	// googleSetupSentence is the setup hint for Google Source.
	googleSetupSentence = "Create an OAuth client ID of type Desktop app at " +
		"https://console.developers.google.com/auth/clients."
)

// ConfigKeyHint names the Git config keys needed when OAuth settings are
// missing.
//
// The hint always mentions the oauthClientId key for the remote URL. Kinds
// whose application comes with a secret also mention oauthClientSecret. Kinds
// that ship a client ID without endpoint URLs also mention oauthAuthURL and
// oauthTokenURL.
//
// Parameters:
//   - gitURL: Git remote URL used as the credential section, such as
//     https://gitlab.example.com.
//
// Returns:
//   - hint: sentence listing the Git config keys to set.
func (d Detected) ConfigKeyHint(gitURL string) string {
	return "Set " + d.configKeys(gitURL) + "."
}

// DefaultUsername returns the Git username to emit when the caller omitted one.
//
// Parameters:
//   - d: detected host. Only Host is consulted.
//
// Returns:
//   - username: x-token-auth for bitbucket.org, otherwise oauth2.
func (d Detected) DefaultUsername() string {
	if d.Host == hostBitbucket {
		return usernameTokenAuth
	}

	return usernameCredentialOAuth2
}

// LoginHintParam returns the authorization-URL query key for a username hint.
//
// Parameters:
//   - d: detected host. Only Kind is consulted.
//
// Returns:
//   - param: login for GitHub and GitHub Enterprise, login_hint for Google
//     Source, otherwise an empty string.
func (d Detected) LoginHintParam() string {
	return d.Kind.LoginHintParam()
}

// SetupHint returns operator instructions when the host has no OAuth
// application configured.
//
// This program ships no application for the host, so the operator registers
// their own. The hint is the whole error they see, which is why it says where
// to register and then names the Git config keys that record the result.
//
// Parameters:
//   - gitURL: Git remote URL used as the credential section, such as
//     https://github.com.
//
// Returns:
//   - hint: where to register an application, then the keys to set.
func (d Detected) SetupHint(gitURL string) string {
	return d.registrationHint() + " Then set " + d.configKeys(gitURL) + "."
}

// SupportsBearer reports whether the host can use HTTP Bearer authentication.
//
// Parameters:
//   - d: detected host. Host, Kind, and Realm are consulted.
//
// Returns:
//   - supported: true for bitbucket.org, codeberg.org, or gitea.com, for kinds
//     Gitea, Forgejo, or Google Source, or for realms Gitea or Forgejo.
func (d Detected) SupportsBearer() bool {
	if bearerHost(d.Host) || bearerRealm(d.Realm) {
		return true
	}

	return d.kindSupportsBearer()
}

// clientIDCanExistWithoutURLs reports kinds that ship a client ID before URLs.
//
// Gitea and Forgejo use the server-registered client ID even when endpoints are
// derived.
//
// Parameters:
//   - d: detected host. Only Kind is consulted.
//
// Returns:
//   - bool: true for Gitea and Forgejo.
func (d Detected) clientIDCanExistWithoutURLs() bool {
	return d.Kind.UsesPublicClientID()
}

// configKeys names the Git config keys an unconfigured host needs.
//
// Parameters:
//   - gitURL: Git remote URL used as the credential section.
//
// Returns:
//   - string: "Git config key" or "Git config keys" followed by the full key
//     names, without a trailing period.
func (d Detected) configKeys(gitURL string) string {
	clientKey := credentialKey(gitURL, keyOAuthClientID)

	switch {
	case d.Kind.NeedsClientSecret():
		return "Git config keys " + clientKey +
			" and " + credentialKey(gitURL, keyOAuthClientSecret)
	case d.clientIDCanExistWithoutURLs():
		return "Git config keys " + clientKey +
			", " + credentialKey(gitURL, keyOAuthAuthURL) +
			", and " + credentialKey(gitURL, keyOAuthTokenURL)
	default:
		return "Git config key " + clientKey
	}
}

// kindSupportsBearer reports kinds whose tokens can be sent as Bearer.
//
// Parameters:
//   - d: detected host. Only Kind is consulted.
//
// Returns:
//   - bool: true for Gitea, Forgejo, and Google Source.
func (d Detected) kindSupportsBearer() bool {
	return d.Kind.SupportsBearerToken()
}

// registrationHint says where an operator registers an application.
//
// gist.github.com has no application form of its own, so every public GitHub
// host is sent to github.com.
//
// Parameters:
//   - d: detected host. Kind and Host are consulted.
//
// Returns:
//   - string: one sentence naming the form and the redirect to register.
func (d Detected) registrationHint() string {
	switch d.Kind.profile().setup {
	case setupGitHub:
		return gitHubSetupHint(hostGitHub)
	case setupGitHubEnterprise:
		return gitHubSetupHint(d.Host)
	case setupGitLab:
		return gitLabSetupHint(d.Host)
	case setupBitbucket:
		return bitbucketSetupSentence
	case setupGoogle:
		return googleSetupSentence
	case setupRegister:
		return registerAppSentence
	default:
		return registerAppSentence
	}
}

// bearerHost reports whether the public host advertises Bearer support.
//
// Parameters:
//   - host: Git hostname, compared as given.
//
// Returns:
//   - bool: true for bitbucket.org, codeberg.org, and gitea.com.
func bearerHost(host string) bool {
	switch host {
	case hostBitbucket, hostCodeberg, hostGitea:
		return true
	default:
		return false
	}
}

// bearerRealm reports whether the realm alone is enough for Bearer support.
//
// Forgejo currently emits a Gitea realm, so both values are accepted.
//
// Parameters:
//   - realm: realm parameter from the WWW-Authenticate header.
//
// Returns:
//   - bool: true for the Gitea and Forgejo realms.
func bearerRealm(realm string) bool {
	switch realm {
	case realmGitea, realmForgejo:
		return true
	default:
		return false
	}
}

// credentialKey builds a credential.<gitURL>.<name> Git config key.
//
// Parameters:
//   - gitURL: Git remote URL used as the credential section.
//   - name: config key suffix, such as oauthClientId.
//
// Returns:
//   - string: credential.<gitURL>.<name>.
func credentialKey(gitURL, name string) string {
	return "credential." + gitURL + "." + name
}

// gitHubSetupHint points operators at the GitHub application form.
//
// Parameters:
//   - host: github.com, or the GitHub Enterprise hostname.
//
// Returns:
//   - string: setup sentence with the form URL and the callback to register.
func gitHubSetupHint(host string) string {
	return "Register an OAuth application at " + schemeHTTPS + "://" + host +
		pathGitHubNewApplication + " with callback URL " + loopbackRedirect + "."
}

// gitLabSetupHint points operators at the GitLab application form.
//
// Confidential is cleared so GitLab expects no client secret, which leaves the
// operator one key to set. The two scopes are the ones Git needs to read and
// push.
//
// Parameters:
//   - host: GitLab hostname inserted into the form URL.
//
// Returns:
//   - string: setup sentence with the form URL and the values to enter.
func gitLabSetupHint(host string) string {
	return "Register an OAuth application at " + schemeHTTPS + "://" + host +
		pathGitLabApplications + " with redirect URI " + loopbackRedirect +
		", Confidential cleared, and the " + scopeReadRepository + " and " +
		scopeWriteRepository + " scopes."
}
