// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package helper

import (
	"context"
	"fmt"
	"io"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/options"
	"github.com/nicholas-fedor/git-credential-oauth/internal/helper"
)

// Helper installs and removes the git credential helper.
type Helper interface {
	// Configure installs the helper.
	//
	// Parameters:
	//   - ctx: cancellation and deadline.
	//   - opts: storage, device flow, and operating system.
	//
	// Returns:
	//   - error: non-nil when git cannot be updated.
	Configure(ctx context.Context, opts helper.Options) error

	// Unconfigure removes the helper.
	//
	// Parameters:
	//   - ctx: cancellation and deadline.
	//   - opts: diagnostic options. The removal plan does not depend on them.
	//
	// Returns:
	//   - error: non-nil when git cannot be updated.
	Unconfigure(ctx context.Context, opts helper.Options) error
}

const (
	// configuredMessage is written to stderr after a successful configure.
	configuredMessage = "configured successfully\n"

	// unconfiguredMessage is written to stderr after a successful unconfigure.
	unconfiguredMessage = "unconfigured successfully\n"
)

// NewConfigureCommand returns the configure command.
//
// Parameters:
//   - ctx: cancellation and deadline for git config updates.
//   - service: helper installer.
//   - stderr: diagnostic stream. git reads standard output as a credential, so
//     the success line must never go there.
//   - opts: parsed storage and device flags.
//
// Returns:
//   - *cobra.Command: the configure command.
func NewConfigureCommand(
	ctx context.Context,
	service Helper,
	stderr io.Writer,
	opts *options.Options,
) *cobra.Command {
	command := &cobra.Command{
		Use:   "configure",
		Short: "Configure as Git credential helper",
		Long: `Register this program as git's credential helper in the global git config.

The existing credential.helper list is replaced, not appended to. It becomes
an empty-string reset, a storage helper, and this program last, so a stored
token is found before a new one is requested. --storage picks the storage
helper; auto chooses wincred on Windows, osxkeychain on macOS, and libsecret
elsewhere.

With --device the helper line is written as "oauth --device", so every later
request uses the device grant.`,
		Example: `# Configure with the storage helper this system prefers.
git-credential-oauth configure

# Record that authentication will use the device flow.
git-credential-oauth --device configure

# Configure with a specific storage helper.
git-credential-oauth configure --storage libsecret`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runConfigure(ctx, service, stderr, opts)
		},
	}
	options.BindStorage(command, opts)

	return command
}

// NewUnconfigureCommand returns the unconfigure command.
//
// Parameters:
//   - ctx: cancellation and deadline for git config updates.
//   - service: helper remover.
//   - stderr: diagnostic stream.
//   - opts: parsed device flags.
//
// Returns:
//   - *cobra.Command: the unconfigure command.
func NewUnconfigureCommand(
	ctx context.Context,
	service Helper,
	stderr io.Writer,
	opts *options.Options,
) *cobra.Command {
	command := &cobra.Command{
		Use:   "unconfigure",
		Short: "Unconfigure as Git credential helper",
		Long: `Remove this program from the global git config.

Every credential.helper value configure can write is removed, whichever
storage helper and grant were chosen, and nothing else. Stored credentials are
left untouched, and nothing is revoked at the forge.`,
		Example: `# Remove the helper from git's configuration.
git-credential-oauth unconfigure`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runUnconfigure(ctx, service, stderr, opts)
		},
	}

	return command
}

// runConfigure installs the helper and writes the success line to stderr.
//
// Parameters:
//   - ctx: context from the cobra command, via Command.Context.
//   - service: helper installer.
//   - stderr: diagnostic stream.
//   - opts: storage and device flag values.
//
// Returns:
//   - error: invalid storage, configure failure, or a write failure.
func runConfigure(
	ctx context.Context,
	service Helper,
	stderr io.Writer,
	opts *options.Options,
) error {
	storage, err := options.ParseStorage(opts.Storage)
	if err != nil {
		return fmt.Errorf("resolve storage option: %w", err)
	}

	err = service.Configure(ctx, helper.Options{
		Storage: storage,
		Device:  opts.Device,
		GOOS:    runtime.GOOS,
	})
	if err != nil {
		return fmt.Errorf("configure git credential helper: %w", err)
	}

	err = options.WriteString(stderr, configuredMessage)
	if err != nil {
		return fmt.Errorf("write configure result: %w", err)
	}

	return nil
}

// runUnconfigure removes the helper and writes the success line to stderr.
//
// Parameters:
//   - ctx: context from the cobra command, via Command.Context.
//   - service: helper remover.
//   - stderr: diagnostic stream.
//   - opts: device flag value.
//
// Returns:
//   - error: unconfigure failure or a write failure.
func runUnconfigure(
	ctx context.Context,
	service Helper,
	stderr io.Writer,
	opts *options.Options,
) error {
	err := service.Unconfigure(ctx, helper.Options{
		Storage: helper.StorageAuto,
		Device:  opts.Device,
		GOOS:    runtime.GOOS,
	})
	if err != nil {
		return fmt.Errorf("unconfigure git credential helper: %w", err)
	}

	err = options.WriteString(stderr, unconfiguredMessage)
	if err != nil {
		return fmt.Errorf("write unconfigure result: %w", err)
	}

	return nil
}
