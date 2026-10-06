// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd"
	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/options"
)

// TestDocRootExposesTheDocumentedSurface checks the tree docgen walks.
//
// DocRoot carries no dependencies, so it is the only tree that can be built
// without wiring services. If it stopped exposing a command, the site's CLI
// reference would silently lose that page rather than fail.
func TestDocRootExposesTheDocumentedSurface(t *testing.T) {
	t.Parallel()

	root := cmd.DocRoot(t.Context())

	require.Equal(t, options.Name, root.Name())

	commands := root.Commands()
	names := make([]string, 0, len(commands))

	for _, command := range commands {
		names = append(names, command.Name())
	}

	assert.Subset(t, names, []string{
		"get",
		"store",
		"erase",
		"configure",
		"unconfigure",
		"version",
		"capability",
	})
}

// TestDocRootCommandsCarryGeneratorMetadata checks what docgen reads is set.
//
// docgen renders Short, Long, and Use. A command missing Long would publish a
// page with an empty description, which is only visible by reading the site.
func TestDocRootCommandsCarryGeneratorMetadata(t *testing.T) {
	t.Parallel()

	for _, command := range cmd.DocRoot(t.Context()).Commands() {
		if !command.IsAvailableCommand() {
			continue
		}

		t.Run(command.Name(), func(t *testing.T) {
			t.Parallel()

			assert.NotEmpty(t, command.Short)
			assert.NotEmpty(t, command.Long)
			assert.NotEmpty(t, command.Use)
		})
	}
}

// TestDocRootOmitsProtocolOnlyOperations checks hidden commands stay hidden.
//
// store and erase exist for gitcredentials(7) to call. Documenting them would
// present git plumbing as something a user runs.
func TestDocRootOmitsProtocolOnlyOperations(t *testing.T) {
	t.Parallel()

	var documented []string

	for _, command := range cmd.DocRoot(t.Context()).Commands() {
		if command.IsAvailableCommand() {
			documented = append(documented, command.Name())
		}
	}

	assert.False(t, slices.Contains(documented, "store"), "store is not hidden")
	assert.False(t, slices.Contains(documented, "erase"), "erase is not hidden")
}
