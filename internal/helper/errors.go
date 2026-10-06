// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package helper

import (
	"errors"

	"github.com/nicholas-fedor/git-credential-oauth/internal/git"
)

var (
	// ErrNilGit is returned when Service.Git is nil.
	ErrNilGit = errors.New("git runner is nil")

	// ErrUnknownStorage is returned when Options.Storage is not recognized.
	ErrUnknownStorage = errors.New("unknown credential storage")
)

// isNotFound reports whether err is a missing git config key.
//
// Only *git.Error with exit code 5 matches. A never-started process and
// any non-git error are returned to the caller unchanged.
//
// Parameters:
//   - err: error returned by GitRunner.Run.
//
// Returns:
//   - bool: true only for git exit code 5.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}

	gitErr, ok := errors.AsType[*git.Error](err)
	if !ok {
		return false
	}

	return gitErr.ExitCode == git.ExitNotFound
}
