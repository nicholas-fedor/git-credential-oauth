// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package capability defines the capability operation, the git credential
// protocol advertisement this program makes.
package capability

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/options"
)

const (
	// capabilityOutput is the git credential capability advertisement.
	capabilityOutput = "version 0\ncapability authtype\n"
)

// NewCommand returns the capability command.
//
// Parameters:
//   - stdout: stream the advertisement is written to.
//
// Returns:
//   - *cobra.Command: the capability command.
func NewCommand(stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:   "capability",
		Short: "Advertise credential helper capabilities",
		Long: `Write the git credential capabilities this helper supports.

The output follows the format of git credential capability, listing version 0
and the authtype capability. With authtype, and --bearer, the helper can return a
Bearer credential instead of a username and password on hosts that accept
one.`,
		Example: `# Ask what this helper supports.
git-credential-oauth capability`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return options.WriteString(stdout, capabilityOutput)
		},
	}

	return command
}
