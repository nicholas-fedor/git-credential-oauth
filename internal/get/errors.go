// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get

import (
	"errors"
)

// MissingConfigError is an incomplete OAuth client for one host.
type MissingConfigError struct {
	// Host is the git credential host.
	Host string

	// GitURL is the protocol and host used for config lookup.
	GitURL string

	// Hint is operator guidance for completing the configuration.
	Hint string
}

// schemeHTTPS is the only transport a remote or configured endpoint may use.
//
// RFC 9700 section 2.6 forbids transmitting an authorization response over an
// unencrypted connection. RFC 8252 section 7.3 permits http for the loopback
// redirect only, and that URL is built locally rather than configured.
const schemeHTTPS = "https"

// ErrMissingConfig is the sentinel for incomplete OAuth configuration.
var ErrMissingConfig = errors.New("missing oauth configuration")

// errNilRegistry is returned when Deps.Forges is nil.
var errNilRegistry = errors.New("forge registry is nil")

// errNilAcquirer is returned when Deps.Auth is nil.
var errNilAcquirer = errors.New("oauth acquirer is nil")

// errInsecureEndpoint is returned when a configured OAuth endpoint is not
// https.
//
// The endpoint receives the authorization code and the access token, so a
// cleartext scheme would send both in the open.
var errInsecureEndpoint = errors.New("oauth endpoint must use https")

// errMalformedEndpoint is returned when a configured OAuth endpoint resolves
// to a URL with no host, or one that does not parse back.
//
// Such a URL would fail later inside the OAuth library with a less useful
// message, so it is refused where the operator's value is first read.
var errMalformedEndpoint = errors.New("oauth endpoint is malformed")

// Error returns the missing-configuration guidance.
//
// Returns:
//   - string: Hint when set, otherwise a host-scoped fallback.
func (e *MissingConfigError) Error() string {
	if e == nil {
		return ErrMissingConfig.Error()
	}

	if e.Hint != "" {
		return e.Hint
	}

	return "missing oauth configuration for host " + e.Host
}

// Unwrap returns the missing-configuration sentinel.
//
// Returns:
//   - error: ErrMissingConfig.
func (e *MissingConfigError) Unwrap() error {
	return ErrMissingConfig
}
