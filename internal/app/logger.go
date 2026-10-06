// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
)

// debugGate reports whether debug logging was requested.
//
// The flag is parsed by cobra during Execute, which is after every service has
// been wired with a logger. A gate that the command tree sets lets one logger
// serve both modes, so the arguments do not have to be parsed twice.
type debugGate struct {
	on atomic.Bool
}

// logger adapts slog to the use-case logger interfaces.
//
// Messages stay static at the call site. args are key-value pairs. When the
// gate is closed every event is dropped, which is the default for a helper that
// git invokes: stderr noise is the user's problem, not the tool's.
type logger struct {
	quiet *slog.Logger
	debug *slog.Logger
	gate  *debugGate
}

// newDebugGate returns a gate that starts disabled.
//
// Returns:
//   - *debugGate: gate defaulting to quiet.
func newDebugGate() *debugGate {
	return &debugGate{}
}

// Enabled reports whether debug logging is on.
//
// Returns:
//   - bool: true when the verbose flag was set.
func (g *debugGate) Enabled() bool {
	return g.on.Load()
}

// Set records the parsed debug flag value.
//
// Parameters:
//   - enabled: value of the verbose flag.
func (g *debugGate) Set(enabled bool) {
	g.on.Store(enabled)
}

// newLogger returns a logger that drops every event.
//
// Returns:
//   - logger: adapter that drops events.
func newLogger() logger {
	return newGatedLogger(io.Discard, newDebugGate())
}

// newDebugLogger returns a logger that writes debug text.
//
// Parameters:
//   - writer: diagnostic stream.
//
// Returns:
//   - logger: adapter that writes debug events.
func newDebugLogger(writer io.Writer) logger {
	gate := newDebugGate()
	gate.Set(true)

	return newGatedLogger(writer, gate)
}

// newGatedLogger returns a logger that follows gate.
//
// Both sinks are built up front and selected per call. Rebuilding the logger
// after parsing would mean every service had to be wired twice.
//
// Parameters:
//   - writer: diagnostic stream used when the gate is open.
//   - gate: switch that selects the debug sink.
//
// Returns:
//   - logger: adapter that follows the gate.
func newGatedLogger(writer io.Writer, gate *debugGate) logger {
	debug := slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
	}))

	return logger{
		quiet: slog.New(slog.DiscardHandler),
		debug: debug,
		gate:  gate,
	}
}

// Debug records a debug event.
//
// Parameters:
//   - ctx: cancellation and request scope.
//   - msg: static message.
//   - args: key-value pairs.
func (adapter logger) Debug(ctx context.Context, msg string, args ...any) {
	//nolint:sloglint // Callers pass static messages and snake_case keys.
	adapter.selectLogger().DebugContext(ctx, msg, args...)
}

// Error records a failed event.
//
// Parameters:
//   - ctx: cancellation and request scope.
//   - msg: static message.
//   - args: key-value pairs.
func (adapter logger) Error(ctx context.Context, msg string, args ...any) {
	//nolint:sloglint // Callers pass static messages and snake_case keys.
	adapter.selectLogger().ErrorContext(ctx, msg, args...)
}

// Info records an informational event.
//
// Parameters:
//   - ctx: cancellation and request scope.
//   - msg: static message.
//   - args: key-value pairs.
func (adapter logger) Info(ctx context.Context, msg string, args ...any) {
	//nolint:sloglint // Callers pass static messages and snake_case keys.
	adapter.selectLogger().InfoContext(ctx, msg, args...)
}

// Warn records a recoverable event.
//
// Parameters:
//   - ctx: cancellation and request scope.
//   - msg: static message.
//   - args: key-value pairs.
func (adapter logger) Warn(ctx context.Context, msg string, args ...any) {
	//nolint:sloglint // Callers pass static messages and snake_case keys.
	adapter.selectLogger().WarnContext(ctx, msg, args...)
}

// selectLogger returns the sink matching the current gate value.
//
// Returns:
//   - *[slog.Logger]: debug sink when the gate is open, discard sink otherwise.
func (adapter logger) selectLogger() *slog.Logger {
	if adapter.gate.Enabled() {
		return adapter.debug
	}

	return adapter.quiet
}
