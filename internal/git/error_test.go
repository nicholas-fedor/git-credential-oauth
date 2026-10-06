// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errUnderlying = errors.New("underlying")

// TestErrorMessageNamesTheCommandAndItsOutput covers every part of the message.
func TestErrorMessageNamesTheCommandAndItsOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  *Error
		want string
	}{
		{err: &Error{ExitCode: 1}, want: "git: exit 1"},
		{
			err:  &Error{Args: []string{"config", "--get", "k"}, ExitCode: 5},
			want: "git config --get k: exit 5",
		},
		{
			err:  &Error{Args: []string{"config"}, ExitCode: 128, Stderr: "  fatal: bad config\n"},
			want: "git config: exit 128: fatal: bad config",
		},
		{
			err:  &Error{ExitCode: ExitNeverStarted, Err: errUnderlying},
			want: "git: exit -1: underlying",
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.err.Error())
	}
}

// TestErrorUnwrapsToTheCause keeps the exec error reachable.
func TestErrorUnwrapsToTheCause(t *testing.T) {
	t.Parallel()

	err := &Error{Err: errUnderlying}
	require.ErrorIs(t, err, errUnderlying)
	assert.NoError(t, (&Error{}).Unwrap())
}
