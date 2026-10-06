// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package helper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/git-credential-oauth/internal/git"
)

// TestIsNotFoundOnlyMatchesExitFive checks which failures are tolerated.
//
// Only git's "nothing to unset" exit is harmless. Any other failure, such as a
// locked or unreadable config file, must stop configure.
func TestIsNotFoundOnlyMatchesExitFive(t *testing.T) {
	t.Parallel()

	assert.False(t, isNotFound(nil))
	assert.False(t, isNotFound(errGitFailed))
	assert.False(t, isNotFound(&git.Error{ExitCode: 1}))
	assert.False(t, isNotFound(&git.Error{ExitCode: git.ExitNeverStarted}))
	assert.True(t, isNotFound(&git.Error{ExitCode: git.ExitNotFound}))
	assert.True(t, isNotFound(fmt.Errorf("wrapped: %w", &git.Error{ExitCode: git.ExitNotFound})))
}
