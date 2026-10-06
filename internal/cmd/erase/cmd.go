// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package erase defines the erase operation, the git credential protocol call
// git makes after an operation fails with a 401.
package erase

import (
	"github.com/spf13/cobra"
)

// NewCommand returns the erase command.
//
// git calls erase when an operation fails with a 401, asking helpers to discard
// the rejected credential. This program has nothing to discard: the token lives
// in the storage helper configured ahead of it, and dropping that is the
// storage helper's decision. Like store, it must stay silent and must not
// change git's view of a failed operation.
//
// Returns:
//   - *cobra.Command: the erase command.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "erase",
		Short:  "Erase credential [called by Git]",
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
