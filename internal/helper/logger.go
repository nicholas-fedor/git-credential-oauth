// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package helper

import (
	"context"
)

// Logger records operational messages for credential-helper changes.
//
// A nil Logger is not called and does not panic.
type Logger interface {
	// Info records a normal progress message.
	//
	// Parameters:
	//   - ctx: cancellation context for the log call.
	//   - msg: static message text.
	//   - args: structured key-value pairs.
	Info(ctx context.Context, msg string, args ...any)

	// Warn records a recovered condition that did not fail the operation.
	//
	// Parameters:
	//   - ctx: cancellation context for the log call.
	//   - msg: static message text.
	//   - args: structured key-value pairs.
	Warn(ctx context.Context, msg string, args ...any)

	// Error records a failure. Callers still return the error.
	//
	// Parameters:
	//   - ctx: cancellation context for the log call.
	//   - msg: static message text.
	//   - args: structured key-value pairs.
	Error(ctx context.Context, msg string, args ...any)
}

const (
	// keyArgs is the log key for a git argument slice.
	keyArgs = "args"

	// keyDevice is the log key for the device-flow flag.
	keyDevice = "device"

	// keyStorage is the log key for the selected storage helper.
	keyStorage = "storage"
)

// logInfo writes an info message when a logger is set.
//
// A nil service or a nil logger is ignored.
//
// Parameters:
//   - ctx: cancellation context passed to the logger.
//   - svc: service that may hold a logger.
//   - msg: static log message.
//   - args: structured key-value pairs.
func logInfo(ctx context.Context, svc *Service, msg string, args ...any) {
	if svc == nil || svc.Log == nil {
		return
	}

	svc.Log.Info(ctx, msg, args...)
}

// logWarn writes a warning when a logger is set.
//
// A nil service or a nil logger is ignored.
//
// Parameters:
//   - ctx: cancellation context passed to the logger.
//   - svc: service that may hold a logger.
//   - msg: static log message.
//   - args: structured key-value pairs.
func logWarn(ctx context.Context, svc *Service, msg string, args ...any) {
	if svc == nil || svc.Log == nil {
		return
	}

	svc.Log.Warn(ctx, msg, args...)
}
