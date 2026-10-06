// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package store_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/store"
)

// TestNewCommand checks the operation git calls is present and silent.
//
// gitcredentials(7) requires an unrecognized operation to be ignored
// silently, and git inherits this program's standard error, so any output here
// would appear in the user's terminal on every authenticated push.
func TestNewCommand(t *testing.T) {
	t.Parallel()

	command := store.NewCommand()

	assert.Equal(t, "store", command.Name())
	assert.True(t, command.Hidden, "store must stay out of the documented surface")
	require.NoError(t, command.Args(command, nil))
	require.Error(t, command.Args(command, []string{"extra"}))
}

// TestNewCommandIsSilent checks nothing reaches either stream.
func TestNewCommandIsSilent(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer

	command := store.NewCommand()
	command.SetOut(&out)
	command.SetErr(&errOut)
	command.SetArgs([]string{})

	require.NoError(t, command.Execute())
	assert.Empty(t, out.String())
	assert.Empty(t, errOut.String())
}
