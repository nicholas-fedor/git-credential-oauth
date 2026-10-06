// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

// Git config keys for one credential context.
//
// Each key is complete, including the credential prefix, because a reader
// passes it straight to git config --get-urlmatch. A reader also supplies the
// URL context, so the credential.<url> part comes from the URL argument
// rather than from these names.
const (
	// OAuthClientID is the OAuth client identifier.
	OAuthClientID = "credential.oauthClientId"

	// OAuthClientSecret is the OAuth client secret.
	OAuthClientSecret = "credential.oauthClientSecret"

	// OAuthAuthURL is the authorization endpoint override.
	OAuthAuthURL = "credential.oauthAuthURL"

	// OAuthTokenURL is the token endpoint override.
	OAuthTokenURL = "credential.oauthTokenURL"

	// OAuthDeviceAuthURL is the device authorization endpoint override.
	OAuthDeviceAuthURL = "credential.oauthDeviceAuthURL"

	// OAuthRedirectURL is the loopback redirect URI override.
	OAuthRedirectURL = "credential.oauthRedirectURL"

	// OAuthScopes is the space-separated scope override.
	OAuthScopes = "credential.oauthScopes"

	// OAuthExpiryMargin is the duration subtracted from a token expiry.
	//
	// The value is a [time.ParseDuration] string.
	OAuthExpiryMargin = "credential.oauthExpiryMargin"
)
