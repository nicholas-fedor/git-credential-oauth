// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAbsentDistinguishesAMissingKeyFromAFailure covers every exit shape.
//
// A missing key is not an error, but a broken config is. Treating a failure as
// absence would silently drop an operator's client ID or endpoint.
func TestAbsentDistinguishesAMissingKeyFromAFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err    error
		name   string
		stdout string
		want   bool
	}{
		{name: "not an exec error", err: errUnderlying, want: false},
		{name: "exit 5", err: &Error{ExitCode: ExitNotFound, Stderr: "anything"}, want: true},
		{name: "exit 1, silent", err: &Error{ExitCode: 1}, want: true},
		{name: "exit 1 with output", err: &Error{ExitCode: 1}, stdout: "value", want: false},
		{name: "exit 1 with an error message", err: &Error{ExitCode: 1, Stderr: "error: bad config"}, want: false},
		{name: "exit 128", err: &Error{ExitCode: 128}, want: false},
		{name: "never started", err: &Error{ExitCode: ExitNeverStarted}, want: false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, absent(tt.err, tt.stdout), tt.name)
	}
}

// TestNewInheritsTheEnvironment checks the default reader.
func TestNewInheritsTheEnvironment(t *testing.T) {
	t.Parallel()

	reader := New("/usr/bin/git")

	assert.Equal(t, "/usr/bin/git", reader.gitPath)
	assert.Nil(t, reader.env)
}
