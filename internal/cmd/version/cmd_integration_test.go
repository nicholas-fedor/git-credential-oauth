// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package version_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/options"
	cmdversion "github.com/nicholas-fedor/git-credential-oauth/internal/cmd/version"
)

// failWriter fails every write so the error path is reachable.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("write output: closed pipe")
}

// TestNewCommand checks the command identity and argument handling.
func TestNewCommand(t *testing.T) {
	t.Parallel()

	command := cmdversion.NewCommand(&bytes.Buffer{}, "dev")

	assert.Equal(t, "version", command.Name())
	require.NoError(t, command.Args(command, nil))
	require.Error(t, command.Args(command, []string{"extra"}))
}

// TestNewCommandPrintsTheProgramName checks the reported version line.
//
// The line is what a user pastes into a bug report, so the program name comes
// from the shared constant rather than a literal here.
func TestNewCommandPrintsTheProgramName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "stamped release",
			version: "1.2.3",
			want:    "git-credential-oauth 1.2.3\n",
		},
		{
			name:    "unstamped development build",
			version: "dev",
			want:    "git-credential-oauth dev\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer

			command := cmdversion.NewCommand(&stdout, tt.version)
			command.SetArgs([]string{})

			require.NoError(t, command.ExecuteContext(t.Context()))
			assert.Equal(t, tt.want, stdout.String())
		})
	}
}

// TestNewCommandReportsAWriteFailure checks a broken pipe is surfaced.
func TestNewCommandReportsAWriteFailure(t *testing.T) {
	t.Parallel()

	command := cmdversion.NewCommand(failWriter{}, "dev")
	command.SetArgs([]string{})

	err := command.ExecuteContext(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, options.ErrWriteOutput)
}
