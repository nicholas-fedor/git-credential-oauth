// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/git"
)

const (
	exampleHost = "https://example.com"
	exampleID   = "test-client-id"
)

func TestReaderGet(t *testing.T) {
	gitPath := isolatedGit(t, exampleConfig())
	reader := git.New(gitPath)

	value, found, err := reader.Get(t.Context(), git.OAuthClientID, exampleHost)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, exampleID, value)

	value, found, err = reader.Get(t.Context(), git.OAuthClientSecret, exampleHost)
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, value)
}

func TestRunnerUnsetMissingKey(t *testing.T) {
	gitPath := isolatedGit(t, exampleConfig())
	runner := git.New(gitPath)

	err := runner.Run(
		t.Context(),
		"config",
		"--global",
		"--unset",
		"credential.https://missing.example.oauthClientId",
	)

	var gitErr *git.Error

	require.ErrorAs(t, err, &gitErr)
	require.Equal(t, git.ExitNotFound, gitErr.ExitCode)
}

func TestReaderBrokenConfig(t *testing.T) {
	gitPath := isolatedGit(t, "this is not a git config\n")
	reader := git.New(gitPath)

	value, found, err := reader.Get(t.Context(), git.OAuthClientID, exampleHost)
	require.Empty(t, value)
	require.False(t, found)
	require.Error(t, err)

	var gitErr *git.Error

	require.ErrorAs(t, err, &gitErr)
	require.NotEqual(t, git.ExitNotFound, gitErr.ExitCode)
	require.NotEqual(t, git.ExitNeverStarted, gitErr.ExitCode)
	require.NotEmpty(t, gitErr.Stderr)
}

func TestNeverStarted(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing-git")
	tests := []struct {
		name string
		path string
	}{
		{name: "missing binary", path: missing},
		{name: "empty path", path: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runner := git.New(tt.path)

			var runErr error

			require.NotPanics(t, func() {
				runErr = runner.Run(t.Context(), "version")
			})
			requireNeverStarted(t, runErr)

			reader := git.New(tt.path)

			var (
				value  string
				found  bool
				getErr error
			)

			require.NotPanics(t, func() {
				value, found, getErr = reader.Get(
					t.Context(),
					git.OAuthClientID,
					exampleHost,
				)
			})
			require.Empty(t, value)
			require.False(t, found)
			requireNeverStarted(t, getErr)
		})
	}
}

// isolatedGit skips when git is not on PATH and points git at a temp config.
func isolatedGit(t *testing.T, body string) string {
	t.Helper()

	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not on PATH")
	}

	dir := t.TempDir()
	cfg := filepath.Join(dir, "gitconfig")
	gitDir := filepath.Join(dir, "gitdir")

	err = os.Mkdir(gitDir, 0o700)
	require.NoError(t, err)

	err = os.WriteFile(cfg, []byte(body), 0o600)
	require.NoError(t, err)

	t.Chdir(dir)
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	t.Setenv("GIT_CONFIG_GLOBAL", cfg)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(dir, "no-system-config"))
	t.Setenv("GIT_DIR", gitDir)
	t.Setenv("GIT_WORK_TREE", dir)
	t.Setenv("GIT_CONFIG_COUNT", "0")
	t.Setenv("GIT_CONFIG_PARAMETERS", "")

	return gitPath
}

func exampleConfig() string {
	return "[credential \"" + exampleHost + "\"]\n\t" +
		"oauthClientId = " + exampleID + "\n"
}

func requireNeverStarted(t *testing.T, err error) {
	t.Helper()

	var gitErr *git.Error

	require.ErrorAs(t, err, &gitErr)
	require.Equal(t, git.ExitNeverStarted, gitErr.ExitCode)
	require.Error(t, gitErr.Unwrap())
}
