// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package helper

import (
	"context"
)

// GitRunner runs git with argument slices that omit the git binary.
type GitRunner interface {
	// Run executes one git invocation.
	//
	// A missing config key is reported as a *git.Error with exit code 5.
	// Failure before the process starts is a different exit code and must
	// not be treated as a missing key.
	//
	// Parameters:
	//   - ctx: cancellation context for the invocation.
	//   - args: git arguments, excluding the git binary.
	//
	// Returns:
	//   - error: non-nil when git cannot be started or exits non-zero.
	Run(ctx context.Context, args ...string) error
}
