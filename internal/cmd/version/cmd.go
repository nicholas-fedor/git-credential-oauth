// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package version defines the version command, which reports the build
// metadata stamped into this binary.
package version

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/options"
)

// NewCommand returns the version command.
//
// The value is a string rather than the whole build metadata because the
// version line is the only thing this command reports.
//
// Parameters:
//   - stdout: stream the version line is written to.
//   - version: stamped version, or the module version for a dev build.
//
// Returns:
//   - *cobra.Command: the version command.
func NewCommand(stdout io.Writer, version string) *cobra.Command {
	command := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Long: `Print the version of the installed binary.

The version is stamped into the binary at link time. A build that was never
stamped reports Go's module version instead, such as a pseudo-version or
(devel), which means the binary came from go install or a local build rather
than a release.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			line := fmt.Sprintf("%s %s\n", options.Name, version)

			return options.WriteString(stdout, line)
		},
	}

	return command
}
