// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import "errors"

var (
	// ErrStateMismatch is returned when the callback state does not match
	// the state sent to the authorization server.
	ErrStateMismatch = errors.New("state mismatch")

	// ErrDeviceUnsupported is returned when device authorization was
	// requested for a provider that does not support it.
	//
	// See https://datatracker.ietf.org/doc/html/rfc8628.
	ErrDeviceUnsupported = errors.New("device authorization is not supported")

	// errNilBrowser is returned when the code grant has no browser.
	errNilBrowser = errors.New("browser is nil")

	// errNilCallback is returned when the code grant has no callback factory.
	errNilCallback = errors.New("callback factory is nil")

	// errNilPrompter is returned when the device grant has no prompter.
	errNilPrompter = errors.New("prompter is nil")

	// errMissingCode is returned when the redirect has no authorization code.
	errMissingCode = errors.New("authorization code is missing")

	// errAuthorization is returned when the redirect carries an OAuth error.
	errAuthorization = errors.New("authorization error")

	// errEmptyState is returned when a nonce produces an empty state.
	errEmptyState = errors.New("generated state is empty")
)
