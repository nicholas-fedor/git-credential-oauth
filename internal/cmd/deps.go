// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"io"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/get"
	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/helper"
)

// DebugGate receives the parsed verbose flag.
//
// The composition root wires a logger before cobra has parsed anything, so the
// flag is reported back through this hook rather than read from [os.Args] a
// second time.
type DebugGate interface {
	// Set records the parsed verbose flag value.
	//
	// Parameters:
	//   - enabled: value of the verbose flag.
	Set(enabled bool)
}

// Dependencies are the process inputs the command tree needs.
//
// The two service fields are contracts declared by the command packages that
// consume them, so a command package depends on the narrowest interface it can
// use and this struct only collects them.
type Dependencies struct {
	Get    get.Getter
	Helper helper.Helper
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Debug  DebugGate
	// Version is the version string the version command reports.
	Version string
}
