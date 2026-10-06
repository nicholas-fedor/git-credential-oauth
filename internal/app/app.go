// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"context"
	"errors"
	"net/http"
	"os/exec"

	"github.com/nicholas-fedor/git-credential-oauth/internal/browser"
	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd"
	"github.com/nicholas-fedor/git-credential-oauth/internal/forge"
	"github.com/nicholas-fedor/git-credential-oauth/internal/get"
	"github.com/nicholas-fedor/git-credential-oauth/internal/git"
	"github.com/nicholas-fedor/git-credential-oauth/internal/helper"
	"github.com/nicholas-fedor/git-credential-oauth/internal/oauth"
)

const (
	// exitOK is a successful command.
	exitOK = 0

	// exitFailure is a usage, runtime, or missing-git failure.
	exitFailure = 1

	// exitMissingConfig is a missing OAuth client configuration.
	exitMissingConfig = 2

	// gitBinary is the executable looked up once per process.
	gitBinary = "git"
)

// The getter and helper services are not listed here. Their contracts live in
// the command packages that consume them, so importing both would need an
// import alias, and the Dependencies literal already fails to compile when a
// service stops satisfying its command.
var (
	_ get.Logger            = logger{}
	_ helper.Logger         = logger{}
	_ oauth.Logger          = logger{}
	_ oauth.Browser         = &browser.Opener{}
	_ oauth.Doer            = &http.Client{}
	_ oauth.CallbackFactory = callbackFactory{}
	_ oauth.Prompter        = terminalPrompter{}
	_ oauth.Acquirer        = &oauth.ConfigAcquirer{}
	_ get.ConfigReader      = &git.Git{}
	_ helper.GitRunner      = &git.Git{}
	_ forge.Registry        = &forge.Table{}
)

// Run executes one git-credential-oauth invocation.
//
// A missing git binary is fatal for every action. Missing OAuth configuration
// is the only path that returns 2. Other errors return 1.
//
// The debug level is gated on a flag the command tree sets after parsing, so
// the logger can be wired into every service before cobra has looked at the
// arguments. That avoids parsing --verbose twice.
//
// Parameters:
//   - ctx: cancellation and deadline, including signal shutdown.
//
// Returns:
//   - int: process status code.
func Run(ctx context.Context) int {
	out := stdStreams()

	gitPath, err := exec.LookPath(gitBinary)
	if err != nil {
		writeDiagnostic(out.Diag, err.Error()+"\n")

		return exitFailure
	}

	debug := newDebugGate()
	log := newGatedLogger(out.Diag, debug)

	root := cmd.NewRoot(ctx, dependencies(gitPath, log, out, debug))

	err = root.ExecuteContext(ctx)
	if err != nil {
		writeDiagnostic(out.Diag, err.Error()+"\n")

		return codeFor(err)
	}

	return exitOK
}

// dependencies assembles the command tree inputs.
//
// Parameters:
//   - gitPath: looked-up git binary.
//   - log: process logger.
//   - out: process streams.
//   - debug: gate the command tree reports the verbose flag to.
//
// Returns:
//   - cmd.Dependencies: wired command inputs.
func dependencies(
	gitPath string,
	log logger,
	out Output,
	debug *debugGate,
) cmd.Dependencies {
	acquirer := newAcquirer(out.Diag, log)

	return cmd.Dependencies{
		Get:     newGetter(gitPath, acquirer, log),
		Helper:  newHelper(gitPath, log),
		Version: newVersion(),
		Stdin:   out.In,
		Stdout:  out.Protocol,
		Stderr:  out.Diag,
		Debug:   debug,
	}
}

// codeFor maps a command error to a process status.
//
// Parameters:
//   - err: error returned by the command tree.
//
// Returns:
//   - int: 2 for missing OAuth configuration, otherwise 1.
func codeFor(err error) int {
	if errors.Is(err, get.ErrMissingConfig) {
		return exitMissingConfig
	}

	return exitFailure
}
