// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package capability_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/capability"
)

// failWriter fails every write so the error path is reachable.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("write output: closed pipe")
}

// advertisement is the exact transcript git parses during discovery.
const advertisement = "version 0\ncapability authtype\n"

// TestNewCommand checks the command identity and argument handling.
func TestNewCommand(t *testing.T) {
	t.Parallel()

	command := capability.NewCommand(&bytes.Buffer{})

	assert.Equal(t, "capability", command.Name())
	require.NoError(t, command.Args(command, nil))
	require.Error(t, command.Args(command, []string{"extra"}))
}

// TestNewCommandAdvertisesCapabilities checks the exact transcript.
//
// git parses this output line by line, so an extra space or a missing newline
// makes the helper look like it advertises nothing.
func TestNewCommandAdvertisesCapabilities(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	command := capability.NewCommand(&stdout)
	command.SetArgs([]string{})

	require.NoError(t, command.ExecuteContext(t.Context()))
	assert.Equal(t, advertisement, stdout.String())
}

// TestNewCommandReportsAWriteFailure checks a broken pipe is surfaced.
func TestNewCommandReportsAWriteFailure(t *testing.T) {
	t.Parallel()

	command := capability.NewCommand(failWriter{})
	command.SetArgs([]string{})

	require.Error(t, command.ExecuteContext(t.Context()))
}
