// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/capability"
	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/erase"
	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/get"
	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/helper"
	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/options"
	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/store"
	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/version"
)

// NewRoot returns the git-credential-oauth command tree wired to deps.
//
// SilenceUsage and SilenceErrors are set so the caller owns usage and error
// text; cobra honors the root values for every subcommand. Every command uses
// RunE and returns errors instead of exiting.
//
// Each subcommand lives in its own package under internal/cmd and exposes a
// single constructor, so the tree is assembled here and nowhere else.
//
// Parameters:
//   - ctx: cancellation and deadline for every subcommand.
//   - deps: process dependencies, including streams and use-case services.
//
// Returns:
//   - *cobra.Command: the root command.
func NewRoot(ctx context.Context, deps Dependencies) *cobra.Command {
	opts := options.New()
	root := &cobra.Command{
		Use:   options.Name,
		Short: "Git credential helper that authenticates to forges using OAuth",
		Long: `git-credential-oauth answers git's credential requests by running an OAuth
flow against the forge that owns the requested host. The token goes back to
git, which hands it to the storage helper configured ahead of this one.

The forge is detected from the requested host, so one installation covers
every forge. Each forge needs an OAuth application you registered yourself,
recorded in credential.<url>.oauthClientId. Nothing is written to standard
output except the credential protocol stream.`,
		Example: `# Install as git's credential helper, behind the platform's storage helper.
git-credential-oauth configure

# Ask git for a credential, running this helper if nothing is stored.
git credential fill < <(printf 'protocol=https\nhost=github.com\n\n')

# Report the build metadata of the installed binary.
git-credential-oauth version`,
		Args: cobra.NoArgs,
		RunE: runHelp,
	}
	silence(root)
	options.BindPersistent(root, opts)

	// The verbose flag is only known after parsing, which happens after every
	// service has been wired. Reporting it here is what lets the composition
	// root keep a single logger for both modes.
	root.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		if deps.Debug != nil {
			deps.Debug.Set(opts.Verbose)
		}

		return nil
	}

	root.AddCommand(get.NewCommand(ctx, deps.Stdin, deps.Stdout, deps.Get, opts))
	root.AddCommand(store.NewCommand())
	root.AddCommand(erase.NewCommand())
	root.AddCommand(helper.NewConfigureCommand(ctx, deps.Helper, deps.Stderr, opts))
	root.AddCommand(helper.NewUnconfigureCommand(ctx, deps.Helper, deps.Stderr, opts))
	root.AddCommand(version.NewCommand(deps.Stdout, deps.Version))
	root.AddCommand(capability.NewCommand(deps.Stdout))

	return root
}

// DocRoot returns the command tree for documentation generation.
//
// tools/docgen reads command metadata only, so the tree carries no
// dependencies: every run function is closed over but never invoked. Building
// the real tree here keeps the documented surface identical to the shipped one.
//
// Parameters:
//   - ctx: context for the tree. The generator never executes it.
//
// Returns:
//   - *cobra.Command: the root command, safe to introspect.
func DocRoot(ctx context.Context) *cobra.Command {
	return NewRoot(ctx, Dependencies{})
}

// runHelp writes command help when no subcommand is given.
//
// Parameters:
//   - command: root command whose help text is written.
//   - args: unused positional arguments.
//
// Returns:
//   - error: non-nil when help cannot be written.
func runHelp(command *cobra.Command, _ []string) error {
	err := command.Help()
	if err != nil {
		return fmt.Errorf("%w: %w", options.ErrWriteOutput, err)
	}

	return nil
}

// silence stops cobra from printing usage or errors on failure.
//
// Cobra checks the root value for every subcommand, so this is set once.
//
// Parameters:
//   - command: command that should stay quiet on error.
func silence(command *cobra.Command) {
	command.SilenceUsage = true
	command.SilenceErrors = true
}
