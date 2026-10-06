// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"github.com/nicholas-fedor/git-credential-oauth/internal/git"
	"github.com/nicholas-fedor/git-credential-oauth/internal/helper"
)

// newHelper wires configure and unconfigure.
//
// Parameters:
//   - gitPath: absolute path of the git binary.
//   - log: diagnostic logger.
//
// Returns:
//   - *helper.Service: helper that mutates git config through GitRunner.
func newHelper(gitPath string, log logger) *helper.Service {
	return &helper.Service{
		Git: git.New(gitPath),
		Log: log,
	}
}
