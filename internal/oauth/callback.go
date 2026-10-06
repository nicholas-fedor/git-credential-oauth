// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"context"
	"fmt"
	"net/url"
)

// Callback receives the authorization-server redirect.
type Callback interface {
	// RedirectURL returns the resolved redirect URI.
	//
	// This is the URI the listener is actually bound to, which may differ
	// from the URI passed to [CallbackFactory.Start] when the port was zero.
	//
	// Returns:
	//   - string: absolute redirect URI.
	RedirectURL() string

	// Wait blocks until the authorization redirect arrives or ctx ends.
	//
	// Duplicate-request handling belongs to the callback implementation.
	// A Wait error, including context cancellation, is returned to the caller.
	//
	// Parameters:
	//   - ctx: cancellation and deadline.
	//
	// Returns:
	//   - url.Values: query values from the redirect.
	//   - error: non-nil when the wait fails or ctx ends.
	Wait(ctx context.Context) (url.Values, error)

	// Close releases the listener.
	//
	// Close is idempotent. Acquire always calls it.
	//
	// Returns:
	//   - error: non-nil when shutdown fails.
	Close() error
}

// CallbackFactory starts a loopback callback listener.
type CallbackFactory interface {
	// Start binds a listener for redirectURL and returns a Callback.
	//
	// Parameters:
	//   - ctx: cancellation and deadline.
	//   - redirectURL: configured redirect URI, possibly with a zero port.
	//
	// Returns:
	//   - Callback: listening callback. The caller must Close it.
	//   - error: non-nil when the listener cannot start.
	Start(ctx context.Context, redirectURL string) (Callback, error)
}

// closeCallback releases callback and wraps a non-nil close error.
//
// Parameters:
//   - callback: listener to close. Nil is a no-op.
//
// Returns:
//   - error: wrapped close failure, or nil.
func closeCallback(callback Callback) error {
	if callback == nil {
		return nil
	}

	err := callback.Close()
	if err != nil {
		return fmt.Errorf("close callback: %w", err)
	}

	return nil
}
