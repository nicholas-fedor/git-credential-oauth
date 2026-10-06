// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package store defines the store operation, the git credential protocol call
// git makes after a successful authentication.
package store

import (
	"github.com/spf13/cobra"
)

// NewCommand returns the store command.
//
// git calls store after a successful authentication to let helpers persist the
// credential. This program holds no credential store of its own: the token is
// written back through the credential.helper chain, and this command exists so
// gitcredentials(7)'s rule that an unrecognized operation is ignored silently
// is honored. It must not print anything, because git inherits this program's
// standard error.
//
// Returns:
//   - *cobra.Command: the store command.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "store",
		Short:  "Store credential [called by Git]",
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE:   runIgnore,
	}
}

// runIgnore succeeds without reading input or writing output.
//
// Parameters:
//   - command: command being run. Unused.
//   - args: unused positional arguments.
//
// Returns:
//   - error: always nil.
func runIgnore(_ *cobra.Command, _ []string) error {
	return nil
}
