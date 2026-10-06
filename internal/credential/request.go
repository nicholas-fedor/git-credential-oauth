// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package credential

// Request is the parsed stdin side of the git credential protocol.
//
// A zero value is a valid empty request. Fields git did not send are empty,
// and every attribute this program does not model is preserved in Extra so
// no input is silently dropped.
type Request struct {
	// Protocol is the transport scheme, such as https.
	Protocol string

	// Host is the server hostname, including a port when present.
	Host string

	// Path is the repository path.
	Path string

	// Username is the login name git already knows, if any.
	Username string

	// Password is the secret git already knows, if any.
	Password string

	// URL is the absolute remote URL, when git sent one.
	URL string

	// WWWAuth holds one value per WWW-Authenticate challenge, in order.
	WWWAuth []string

	// Capability holds one value per advertised capability, in order.
	Capability []string

	// OAuthRefreshToken is a refresh token from a previous authorization.
	OAuthRefreshToken string

	// Extra holds attributes this program does not model.
	//
	// A repeated single-valued key keeps only its last value. A multi-valued
	// key keeps every occurrence, in input order.
	Extra []Pair
}

// emptyRequest returns a request with no attributes set.
//
// Returns:
//   - Request: zero request with no extra attributes.
func emptyRequest() Request {
	return Request{
		Protocol:          "",
		Host:              "",
		Path:              "",
		Username:          "",
		Password:          "",
		URL:               "",
		WWWAuth:           nil,
		Capability:        nil,
		OAuthRefreshToken: "",
		Extra:             nil,
	}
}
