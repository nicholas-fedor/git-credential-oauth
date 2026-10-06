// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package erase_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/erase"
)

// TestNewCommand checks the operation git calls is present and silent.
//
// Like store, it must not change git's view of a failed operation, so a
// rejection cannot turn into a reported success.
func TestNewCommand(t *testing.T) {
	t.Parallel()

	command := erase.NewCommand()

	assert.Equal(t, "erase", command.Name())
	assert.True(t, command.Hidden, "erase must stay out of the documented surface")
	require.NoError(t, command.Args(command, nil))
	require.Error(t, command.Args(command, []string{"extra"}))
}

// TestNewCommandIsSilent checks nothing reaches either stream.
func TestNewCommandIsSilent(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer

	command := erase.NewCommand()
	command.SetOut(&out)
	command.SetErr(&errOut)
	command.SetArgs([]string{})

	require.NoError(t, command.Execute())
	assert.Empty(t, out.String())
	assert.Empty(t, errOut.String())
}
