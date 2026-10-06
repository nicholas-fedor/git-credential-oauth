// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package credential

// Pair is one credential protocol attribute.
//
// Git calls these attributes. A key is the text before the first equals sign
// on a line, and the value is everything after it, so a value may itself
// contain an equals sign.
type Pair struct {
	// Key is the attribute name.
	Key string

	// Value is the attribute value. It may be empty and may contain '='.
	Value string
}

const (
	// Protocol is the transport scheme attribute, such as https.
	Protocol = "protocol"

	// Host is the server hostname attribute, including a port when present.
	Host = "host"

	// Path is the repository path attribute.
	Path = "path"

	// Username is the login name attribute.
	Username = "username"

	// Password is the secret attribute.
	Password = "password"

	// URL is the absolute remote URL attribute.
	URL = "url"

	// OAuthRefreshToken is the OAuth refresh token attribute.
	OAuthRefreshToken = "oauth_refresh_token"

	// PasswordExpiryUTC is the password expiry attribute, in Unix seconds.
	//
	// Git reads a value of 0 as never expiring, so a zero expiry omits the key.
	PasswordExpiryUTC = "password_expiry_utc"

	// Authtype is the authentication scheme attribute.
	//
	// The same text names the capability git advertises, so this constant is
	// both the emitted key and the capability value to test for.
	Authtype = "authtype"

	// Credential is the authtype credential attribute.
	Credential = "credential"
)

const (
	// WWWAuth is the multi-valued WWW-Authenticate attribute.
	//
	// The empty brackets are part of the key, so this constant is both the
	// parsed key and the emitted key.
	WWWAuth = "wwwauth[]"

	// Capability is the multi-valued capability attribute.
	//
	// As with WWWAuth, the empty brackets are part of the key.
	Capability = "capability[]"

	// arraySuffix marks a key as multi-valued when parsing a request.
	arraySuffix = "[]"
)
