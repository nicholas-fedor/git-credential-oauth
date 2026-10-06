// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get

import "context"

// Logger records diagnostic events for a credential get.
//
// Messages are static. args are key-value pairs, as with log/slog.
// A nil Deps.Log is a no-op.
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
