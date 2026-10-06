// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"errors"
	"io/fs"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommandCopiesTheEnvironment checks a caller's slice is not shared.
func TestCommandCopiesTheEnvironment(t *testing.T) {
	t.Parallel()

	env := []string{"A=1"}
	cmd := command(t.Context(), "/bin/true", env, []string{"x"})
	env[0] = "A=changed"

	assert.Equal(t, []string{"A=1"}, cmd.Env)
	assert.Equal(t, []string{"/bin/true", "x"}, cmd.Args)

	inherited := command(t.Context(), "/bin/true", nil, nil)
	assert.Nil(t, inherited.Env, "a nil environment inherits the process's")
}

// TestExitCodeOfAProcessThatNeverRan covers a missing binary.
func TestExitCodeOfAProcessThatNeverRan(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ExitNeverStarted, exitCode(nil))
}

// TestInvokeReportsAMissingBinary keeps the cause and the never-started code.
func TestInvokeReportsAMissingBinary(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "no-such-git")

	_, err := invoke(t.Context(), missing, nil, []string{"version"})

	gitErr, ok := errors.AsType[*Error](err)
	require.True(t, ok, "error %v", err)
	assert.Equal(t, ExitNeverStarted, gitErr.ExitCode)
	assert.Equal(t, []string{"version"}, gitErr.Args)

	// Unix reports a missing absolute path as a missing file. Windows looks for
	// an executable extension first and reports it as not found instead.
	assert.True(t,
		errors.Is(err, fs.ErrNotExist) || errors.Is(err, exec.ErrNotFound),
		"cause %v", err)
}

// TestInvokeCapturesOutputAndExitStatus runs the real git binary.
func TestInvokeCapturesOutputAndExitStatus(t *testing.T) {
	t.Parallel()

	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}

	stdout, err := invoke(t.Context(), gitPath, nil, []string{"version"})
	require.NoError(t, err)
	assert.Contains(t, stdout, "git version")

	_, err = invoke(t.Context(), gitPath, nil, []string{"no-such-subcommand"})

	gitErr, ok := errors.AsType[*Error](err)
	require.True(t, ok, "error %v", err)
	assert.Positive(t, gitErr.ExitCode)
	assert.NotEmpty(t, gitErr.Stderr)
}
