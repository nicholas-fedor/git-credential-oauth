// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package browser

import (
	"context"
	"errors"
	"fmt"
	"runtime"
)

// Opener opens a URL with the platform browser command.
type Opener struct {
	// LookPath resolves a command name on PATH.
	//
	// Nil uses [exec.LookPath].
	LookPath func(string) (string, error)

	// Run starts the command and waits for it to exit.
	//
	// Nil uses [exec.CommandContext].
	Run func(ctx context.Context, name string, args ...string) error

	// GOOS is the operating system name used to select the command.
	//
	// Empty uses [runtime.GOOS].
	GOOS string
}

// ErrNoBrowser is returned when the platform browser command is not on PATH.
var ErrNoBrowser = errors.New("browser command not found")

// Open starts the platform browser for targetURL.
//
// A nil LookPath uses [exec.LookPath]. A nil Run uses [exec.CommandContext].
// An empty GOOS uses [runtime.GOOS]. A missing command returns ErrNoBrowser.
// A start failure is a different error.
//
// Parameters:
//   - ctx: cancellation and deadline for the browser process.
//   - targetURL: URL or path to open.
//
// Returns:
//   - error: ErrNoBrowser when the command is missing, or a run failure.
func (o *Opener) Open(ctx context.Context, targetURL string) error {
	commandName, commandArgs := Command(o.platform(), targetURL)

	// Confirm the command exists. Execute the name, not a resolved path.
	_, err := o.lookPath()(commandName)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNoBrowser, err)
	}

	err = o.run()(ctx, commandName, commandArgs...)
	if err != nil {
		return fmt.Errorf("run browser %s: %w", commandName, err)
	}

	return nil
}

// lookPath returns the configured resolver, or the exec default.
//
// Parameters:
//   - o: opener whose LookPath is read.
//
// Returns:
//   - func(string) (string, error): configured resolver, or defaultLookPath.
func (o *Opener) lookPath() func(string) (string, error) {
	if o.LookPath != nil {
		return o.LookPath
	}

	return defaultLookPath
}

// platform returns the configured GOOS, or the process GOOS.
//
// Parameters:
//   - o: opener whose GOOS is read.
//
// Returns:
//   - string: configured GOOS, or [runtime.GOOS].
func (o *Opener) platform() string {
	if o.GOOS != "" {
		return o.GOOS
	}

	return runtime.GOOS
}

// run returns the configured starter, or the exec default.
//
// Parameters:
//   - o: opener whose Run is read.
//
// Returns:
//   - func(ctx [context.Context], name string, args ...string) error:
//     configured starter, or defaultRun.
func (o *Opener) run() func(
	ctx context.Context,
	name string,
	args ...string,
) error {
	if o.Run != nil {
		return o.Run
	}

	return defaultRun
}
