// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/git-credential-oauth/internal/version"
)

// TestNewVersionReportsTheBuildVersion checks the value commands print.
func TestNewVersionReportsTheBuildVersion(t *testing.T) {
	t.Parallel()

	assert.Equal(t, version.GetVersion(), newVersion())
	assert.NotEmpty(t, newVersion())
}
