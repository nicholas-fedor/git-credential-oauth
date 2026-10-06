// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"context"

	"golang.org/x/oauth2"
)

// Input selects how Acquire obtains a token.
type Input struct {
	// RefreshToken is an existing refresh token.
	//
	// When non-empty, Acquire tries a refresh grant before any interactive
	// grant. Refresh failure is logged and does not fail the call by itself.
	RefreshToken string

	// AuthURLSuffix is appended verbatim to the authorization URL.
	//
	// Include the leading ampersand. The device grant ignores it.
	AuthURLSuffix string

	// Device requests the device authorization grant.
	Device bool

	// DeviceFlow reports that the provider supports device authorization.
	//
	// Device without DeviceFlow returns [ErrDeviceUnsupported].
	DeviceFlow bool

	// PKCE is informational.
	//
	// The authorization-code grant always uses S256, regardless of this field.
	PKCE bool
}

// Acquirer obtains an OAuth token for a configured client.
type Acquirer interface {
	// Acquire returns a token using refresh, device, or authorization code.
	//
	// Parameters:
	//   - ctx: cancellation and deadline.
	//   - cfg: client and endpoint configuration.
	//   - in: grant selection and refresh token.
	//
	// Returns:
	//   - Token: acquired credentials.
	//   - error: non-nil when no grant succeeds.
	Acquire(ctx context.Context, cfg oauth2.Config, in Input) (Token, error)
}

// ConfigAcquirer obtains tokens from an [oauth2.Config].
//
// A nil Log is a no-op. Browser and Callback are required for the
// authorization-code grant. Prompter is required for the device grant.
// Doer, when non-nil, is injected as the oauth2 HTTP client.
type ConfigAcquirer struct {
	// Browser opens the authorization URL. Required for the code grant.
	Browser Browser

	// Callback starts the loopback redirect listener. Required for the code grant.
	Callback CallbackFactory

	// Prompter displays the device user code. Required for the device grant.
	Prompter Prompter

	// Doer is the HTTP client for token requests.
	//
	// Nil uses the default client.
	Doer Doer

	// Log receives diagnostic events. Nil is a no-op.
	Log Logger

	// Nonce generates CSRF state. Nil uses [oauth2.GenerateVerifier].
	Nonce Nonce
}

// Logger records diagnostic events for a grant.
//
// Messages are static. args are key-value pairs, as with log/slog.
type Logger interface {
	// Debug records a debug event.
	//
	// Parameters:
	//   - ctx: cancellation and request scope.
	//   - msg: static message.
	//   - args: key-value pairs.
	Debug(ctx context.Context, msg string, args ...any)

	// Info records an informational event.
	//
	// Parameters:
	//   - ctx: cancellation and request scope.
	//   - msg: static message.
	//   - args: key-value pairs.
	Info(ctx context.Context, msg string, args ...any)

	// Warn records a recoverable event.
	//
	// Parameters:
	//   - ctx: cancellation and request scope.
	//   - msg: static message.
	//   - args: key-value pairs.
	Warn(ctx context.Context, msg string, args ...any)

	// Error records a failed event.
	//
	// Parameters:
	//   - ctx: cancellation and request scope.
	//   - msg: static message.
	//   - args: key-value pairs.
	Error(ctx context.Context, msg string, args ...any)
}

// nopLogger discards log events so a nil Log cannot panic.
type nopLogger struct{}

// Compile-time checks for the acquirer and the no-op logger.
var (
	_ Acquirer = (*ConfigAcquirer)(nil)
	_ Logger   = nopLogger{}
)

// Debug discards a debug event.
func (nopLogger) Debug(context.Context, string, ...any) {}

// Error discards an error event.
func (nopLogger) Error(context.Context, string, ...any) {}

// Info discards an informational event.
func (nopLogger) Info(context.Context, string, ...any) {}

// Warn discards a warning event.
func (nopLogger) Warn(context.Context, string, ...any) {}

// logger returns Log, or a no-op logger when Log is nil.
//
// Parameters:
//   - a: acquirer whose logger is read. Nil uses the no-op logger.
//
// Returns:
//   - Logger: configured logger, or a discard logger.
//
//nolint:ireturn // nil logger becomes the nop implementation
func (a *ConfigAcquirer) logger() Logger {
	if a == nil || a.Log == nil {
		return nopLogger{}
	}

	return a.Log
}
