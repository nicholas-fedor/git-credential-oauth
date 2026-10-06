// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package browser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultLookPathRejectsMissingCommand checks the exec default is wired.
func TestDefaultLookPathRejectsMissingCommand(t *testing.T) {
	t.Parallel()

	path, err := defaultLookPath("git-credential-oauth-no-such-command")
	require.Error(t, err)
	assert.Empty(t, path)
}

// TestDefaultLookPathResolvesKnownCommand checks a real command is found.
func TestDefaultLookPathResolvesKnownCommand(t *testing.T) {
	t.Parallel()

	// A command that is present on every platform Go supports.
	path, err := defaultLookPath("go")
	if err != nil {
		t.Skipf("go is not on PATH: %v", err)
	}

	assert.NotEmpty(t, path)
}

// TestDefaultRunStartsAProcess checks the exec default is wired.
func TestDefaultRunStartsAProcess(t *testing.T) {
	t.Parallel()

	err := defaultRun(t.Context(), "go", "version")
	require.NoError(t, err)
}

// TestDefaultRunSurfacesNonZeroExit checks a failing command is reported.
func TestDefaultRunSurfacesNonZeroExit(t *testing.T) {
	t.Parallel()

	err := defaultRun(t.Context(), "go", "this-is-not-a-subcommand")
	require.Error(t, err)
}
