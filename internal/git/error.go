// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"strconv"
	"strings"
)

// Error is a failed git invocation.
//
// ExitCode is ExitNeverStarted when the process did not start. Callers must
// not treat that value, or any exit other than ExitNotFound, as a missing key.
type Error struct {
	Err      error
	Stderr   string
	Args     []string
	ExitCode int
}

const (
	// ExitNotFound is the git exit code for unsetting a key that does not exist.
	ExitNotFound = 5

	// ExitNeverStarted is reported when git fails before the process starts.
	ExitNeverStarted = -1

	// exitKeyAbsent is the git config --get-urlmatch exit code for a missing
	// value.
	exitKeyAbsent = 1
)

// Error returns a description of the failed git command.
//
// The message includes the arguments, exit code, trimmed stderr, and the
// wrapped error when those parts are present.
//
// Parameters:
//   - e: the git command error.
//
// Returns:
//   - string: args, exit code, stderr, and the wrapped error.
func (e *Error) Error() string {
	msg := "git"
	if len(e.Args) > 0 {
		msg += " " + strings.Join(e.Args, " ")
	}

	msg += ": exit " + strconv.Itoa(e.ExitCode)

	stderr := strings.TrimSpace(e.Stderr)
	if stderr != "" {
		msg += ": " + stderr
	}

	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}

	return msg
}

// Unwrap returns the underlying exec or context error.
//
// Parameters:
//   - e: the git command error.
//
// Returns:
//   - error: the wrapped error, or nil when none was recorded.
func (e *Error) Unwrap() error {
	return e.Err
}
