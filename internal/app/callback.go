// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/nicholas-fedor/git-credential-oauth/internal/callback"
	"github.com/nicholas-fedor/git-credential-oauth/internal/oauth"
	"github.com/nicholas-fedor/git-credential-oauth/internal/version"
)

// callbackFactory adapts callback.Factory to oauth.CallbackFactory.
//
// Factory.Start returns a concrete server. The oauth interface requires the
// Callback interface, so the adapter converts the result.
//
// The adapter lives here rather than in either package because callback must
// not import oauth, which would invert the dependency and make the loopback
// server depend on the grant it serves.
type callbackFactory struct {
	inner *callback.Factory
}

// errStartCallback is returned when the loopback listener cannot start.
var errStartCallback = errors.New("start oauth callback")

// newCallbackFactory returns a factory stamped with the build version.
//
// The version reaches the loopback success page, so a user reporting a problem
// can see which build produced it.
//
// Returns:
//   - callbackFactory: adapter whose Start returns an oauth callback.
func newCallbackFactory() callbackFactory {
	return callbackFactory{
		inner: &callback.Factory{
			Version: version.GetVersion(),
		},
	}
}

// Start binds a loopback listener and returns it as an oauth callback.
//
// A start failure returns a nil interface, not a typed nil server, so a caller
// comparing the result against nil sees the failure rather than a live value
// that panics on use.
//
// Parameters:
//   - ctx: cancellation and deadline for the bind.
//   - redirectURL: configured redirect, or empty for an ephemeral port.
//
// Returns:
//   - oauth.Callback: running listener.
//   - error: non-nil when the listener cannot start.
//
//nolint:ireturn // adapter must return the consumer interface.
func (factory callbackFactory) Start(
	ctx context.Context,
	redirectURL string,
) (oauth.Callback, error) {
	server, err := factory.inner.Start(ctx, redirectURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errStartCallback, err)
	}

	return server, nil
}
