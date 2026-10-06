// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"slices"
)

// command builds a git process that has not been started.
//
// A nil env inherits the process environment. The caller attaches output
// streams before starting the process.
//
// Parameters:
//   - ctx: cancellation context for the process.
//   - gitPath: git binary path. An empty path fails before start.
//   - env: replacement environment, or nil to inherit.
//   - args: git arguments, excluding the binary path.
//
// Returns:
//   - *[exec.Cmd]: configured command that has not been started.
func command(
	ctx context.Context,
	gitPath string,
	env, args []string,
) *exec.Cmd {
	cmd := exec.CommandContext(ctx, gitPath, args...)
	if env != nil {
		cmd.Env = slices.Clone(env)
	}

	return cmd
}

// invoke runs git and captures both output streams.
//
// A nil error means git exited zero. Failure before start is an Error with
// ExitNeverStarted. ProcessState is never dereferenced when it is nil.
//
// Parameters:
//   - ctx: cancellation context for the process.
//   - gitPath: git binary path.
//   - env: replacement environment, or nil to inherit.
//   - args: git arguments, excluding the binary path.
//
// Returns:
//   - string: captured standard output, which may be empty.
//   - error: an [Error] when git does not start or exits non-zero, or nil.
func invoke(
	ctx context.Context,
	gitPath string,
	env, args []string,
) (string, error) {
	cmd := command(ctx, gitPath, env, args)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return stdout.String(), commandError(
			args, stderr.String(), cmd.ProcessState, err,
		)
	}

	return stdout.String(), nil
}

// commandError records a git failure without touching a nil process state.
//
// Parameters:
//   - args: git arguments, excluding the binary path.
//   - stderr: captured standard error.
//   - state: process state, or nil when the process did not start.
//   - err: error returned by the exec package.
//
// Returns:
//   - *[Error]: non-nil failure describing the invocation.
func commandError(
	args []string,
	stderr string,
	state *os.ProcessState,
	err error,
) *Error {
	return &Error{
		Args:     slices.Clone(args),
		ExitCode: exitCode(state),
		Stderr:   stderr,
		Err:      err,
	}
}

// exitCode returns the process exit code.
//
// A nil state means the process never started. ExitCode is not called on
// a nil state.
//
// Parameters:
//   - state: process state, or nil when the process did not start.
//
// Returns:
//   - int: ExitNeverStarted, or the process exit code.
func exitCode(state *os.ProcessState) int {
	if state == nil {
		return ExitNeverStarted
	}

	return state.ExitCode()
}
