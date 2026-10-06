// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package browser

import (
	"context"
	"fmt"
	"os/exec"
)

// defaultLookPath resolves name with [exec.LookPath].
//
// Parameters:
//   - name: executable name.
//
// Returns:
//   - string: resolved path.
//   - error: name is not on PATH.
func defaultLookPath(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("look up %s: %w", name, err)
	}

	return path, nil
}

// defaultRun starts name with args and waits for it to exit.
//
// Parameters:
//   - ctx: cancellation and deadline.
//   - name: executable path.
//   - args: command arguments.
//
// Returns:
//   - error: the process failed to start or exited non-zero.
func defaultRun(ctx context.Context, name string, args ...string) error {
	err := exec.CommandContext(
		ctx, name, args...,
	).Run()
	if err != nil {
		return fmt.Errorf("run browser command: %w", err)
	}

	return nil
}
