// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// acquireDevice runs the RFC 8628 device authorization grant.
//
// DeviceAuth is not called when the provider does not support the grant.
// PromptDevice errors abort the grant before polling for a token.
//
// Parameters:
//   - ctx: cancellation and deadline.
//   - cfg: OAuth client configuration.
//   - in: grant selection. DeviceFlow false returns ErrDeviceUnsupported.
//
// Returns:
//   - Token: access token from the device grant.
//   - error: unsupported device flow, prompt failure, or token endpoint
//     failure.
func (a *ConfigAcquirer) acquireDevice(
	ctx context.Context,
	cfg oauth2.Config,
	in Input,
) (Token, error) {
	if !in.DeviceFlow {
		return Token{}, ErrDeviceUnsupported
	}

	if a == nil || a.Prompter == nil {
		return Token{}, errNilPrompter
	}

	ctx = withDoer(ctx, a.Doer)

	auth, err := cfg.DeviceAuth(ctx)
	if err != nil {
		return Token{}, fmt.Errorf("device auth: %w", err)
	}

	err = a.Prompter.PromptDevice(
		ctx,
		auth.UserCode,
		auth.VerificationURI,
		auth.VerificationURIComplete,
	)
	if err != nil {
		return Token{}, fmt.Errorf("prompt device: %w", err)
	}

	token, err := cfg.DeviceAccessToken(ctx, auth)
	if err != nil {
		return Token{}, fmt.Errorf("device access token: %w", err)
	}

	return FromOAuth2(token), nil
}
