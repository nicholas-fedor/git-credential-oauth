// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/options"
	"github.com/nicholas-fedor/git-credential-oauth/internal/credential"
	"github.com/nicholas-fedor/git-credential-oauth/internal/get"
)

// Getter resolves a git credential request.
type Getter interface {
	// Get returns the credential response for req.
	//
	// Parameters:
	//   - ctx: cancellation and deadline.
	//   - req: parsed credential request.
	//   - opts: bearer and device-flow options.
	//
	// Returns:
	//   - credential.Response: ordered protocol response.
	//   - error: non-nil when the request cannot be satisfied.
	Get(
		ctx context.Context,
		req credential.Request,
		opts get.Options,
	) (credential.Response, error)
}

// NewCommand returns the get command.
//
// Parameters:
//   - ctx: cancellation and deadline for the credential request.
//   - stdin: stream git writes the credential request to.
//   - stdout: stream the credential response is written to.
//   - getter: service that satisfies the request.
//   - opts: parsed device and bearer flags.
//
// Returns:
//   - *cobra.Command: the get command.
func NewCommand(
	ctx context.Context,
	stdin io.Reader,
	stdout io.Writer,
	getter Getter,
	opts *options.Options,
) *cobra.Command {
	command := &cobra.Command{
		Use:   "get",
		Short: "Generate credential [called by Git]",
		Long: `Read a credential request from standard input and write the resolved
credential to standard output in the git credential protocol format.

git invokes this operation itself after the helper is configured. The
response is the only thing written to standard output; diagnostics go to
standard error so a failure never corrupts the protocol stream.`,
		Example: `# Run the helper directly, with diagnostics on standard error.
printf 'protocol=https\nhost=github.com\n\n' | git-credential-oauth --verbose get

# Use the device grant instead of a browser.
printf 'protocol=https\nhost=github.com\n\n' | git-credential-oauth --device get`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return run(ctx, stdin, stdout, getter, opts)
		},
	}

	return command
}

// run parses stdin, resolves a credential, and writes the response.
//
// Parameters:
//   - ctx: context from the cobra command, via Command.Context.
//   - stdin: stream holding the credential request.
//   - stdout: stream the response is written to.
//   - getter: service that satisfies the request.
//   - opts: device and bearer flag values.
//
// Returns:
//   - error: parse, get, or write failure. It is not discarded.
func run(
	ctx context.Context,
	stdin io.Reader,
	stdout io.Writer,
	getter Getter,
	opts *options.Options,
) error {
	req, err := credential.Parse(stdin)
	if err != nil {
		return fmt.Errorf("parse credential request: %w", err)
	}

	resp, err := getter.Get(ctx, req, get.Options{
		UseBearer: opts.Bearer,
		Device:    opts.Device,
	})
	if err != nil {
		return fmt.Errorf("get credential: %w", err)
	}

	err = resp.Write(stdout)
	if err != nil {
		return fmt.Errorf("write credential response: %w", err)
	}

	return nil
}
