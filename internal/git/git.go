// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"context"
	"errors"
	"strings"
)

// Git runs the git binary for both reading configuration and applying it.
//
// There is one type rather than a reader and a runner because both do the same
// thing: start git, capture both streams, and turn a failure into an [Error].
// Splitting them meant two constructors and two copies of the same contract
// that a caller could wire inconsistently.
//
// An empty path is not resolved here. The composition root locates the binary
// once, so a missing git is reported before any credential work starts.
type Git struct {
	// gitPath is the git binary executed by Get and Run.
	gitPath string

	// env, when non-nil, replaces the process environment for git.
	// A nil env inherits the current process environment.
	env []string
}

// New returns a Git bound to gitPath.
//
// Parameters:
//   - gitPath: path to the git binary.
//
// Returns:
//   - *Git: git client bound to gitPath.
func New(gitPath string) *Git {
	return &Git{
		gitPath: gitPath,
		env:     nil,
	}
}

// Get returns the URL-matched git config value for key.
//
// It runs git config --get-urlmatch. Exit code 5, exit code 1 with empty
// output, and empty stdout are a missing key. Any other failure is returned so
// a broken git is never reported as an unset key, which would otherwise start
// an interactive authorization the operator did not ask for.
//
// Parameters:
//   - ctx: cancellation context for the git process.
//   - key: git config key, such as credential.oauthClientId.
//   - rawURL: URL used for urlmatch, such as https://example.com.
//
// Returns:
//   - value: trimmed stdout when the key is set.
//   - found: true when a non-empty value was read.
//   - error: non-nil when git failed for a reason other than a missing key.
func (g *Git) Get(
	ctx context.Context,
	key, rawURL string,
) (string, bool, error) {
	stdout, runErr := invoke(ctx, g.gitPath, g.env, []string{
		"config",
		"--get-urlmatch",
		key,
		rawURL,
	})
	if runErr != nil {
		if absent(runErr, stdout) {
			return "", false, nil
		}

		return "", false, runErr
	}

	value := strings.TrimSpace(stdout)
	if value == "" {
		return "", false, nil
	}

	return value, true, nil
}

// Run executes git with args and discards its output.
//
// args excludes the git binary. A nil error means git exited zero. Exit code 5
// is reported as an [Error] like any other failure, and the caller decides
// whether to ignore it.
//
// Parameters:
//   - ctx: cancellation context for the git process.
//   - args: git arguments, excluding the binary path.
//
// Returns:
//   - error: *Error when git does not start or exits non-zero, or nil.
func (g *Git) Run(ctx context.Context, args ...string) error {
	_, err := invoke(ctx, g.gitPath, nil, args)

	return err
}

// absent reports whether git config treated the key as unset.
//
// git config --unset exits 5 when the key does not exist. git config
// --get-urlmatch exits 1 with empty output for the same condition. A non-empty
// stderr or any other exit code is a broken git, not an unset key, so it is not
// swallowed here.
//
// Parameters:
//   - runErr: error returned by invoke.
//   - stdout: captured standard output from that invocation.
//
// Returns:
//   - bool: true when the key should be reported as not found.
func absent(runErr error, stdout string) bool {
	gitErr, matched := errors.AsType[*Error](runErr)
	if !matched {
		return false
	}

	if gitErr.ExitCode == ExitNotFound {
		return true
	}

	if gitErr.ExitCode != exitKeyAbsent {
		return false
	}

	if strings.TrimSpace(stdout) != "" {
		return false
	}

	return strings.TrimSpace(gitErr.Stderr) == ""
}
