// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/git-credential-oauth/internal/git"
)

// TestNewHelperUsesTheGivenGit checks configure runs the git that was found.
func TestNewHelperUsesTheGivenGit(t *testing.T) {
	t.Parallel()

	svc := newHelper("/usr/bin/git", newLogger())

	assert.IsType(t, &git.Git{}, svc.Git)
	assert.NotNil(t, svc.Log)
}
