// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package options

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/git-credential-oauth/internal/helper"
)

// Options holds flag values for one command execution.
type Options struct {
	// Storage is the credential storage helper configure installs.
	Storage string
	// Device selects the device authorization grant.
	Device bool
	// Bearer prefers Bearer authentication for supported hosts.
	Bearer bool
	// Verbose logs debug information to stderr.
	Verbose bool
}

const (
	// Name is the program name.
	//
	// It is the cobra root Use value and the prefix of the version line, so
	// both stay in step with the installed binary. It lives here rather than
	// beside the build metadata because the command layer cannot import two
	// packages that both declare the name "version".
	Name = "git-credential-oauth"

	// flagVerbose is the persistent debug flag name.
	flagVerbose = "verbose"

	// flagVerboseShort is the short form of the debug flag.
	flagVerboseShort = "v"

	// flagDevice is the persistent device-flow flag name.
	flagDevice = "device"

	// flagBearer is the persistent bearer flag name.
	flagBearer = "bearer"

	// flagStorage is the configure storage flag name.
	flagStorage = "storage"
)

// errUnsupportedStorage is returned when --storage is not an accepted value.
var errUnsupportedStorage = errors.New("storage is not supported")

// New returns flag defaults before cobra parses arguments.
//
// Returns:
//   - *Options: zero device, bearer, and verbose, storage auto.
func New() *Options {
	return &Options{
		Device:  false,
		Bearer:  false,
		Verbose: false,
		Storage: string(helper.StorageAuto),
	}
}

// BindPersistent registers the flags shared by every command.
//
// Parameters:
//   - command: root command that owns the persistent flag set.
//   - opts: destination for device, bearer, and verbose.
func BindPersistent(command *cobra.Command, opts *Options) {
	command.PersistentFlags().BoolVarP(
		&opts.Verbose,
		flagVerbose,
		flagVerboseShort,
		false,
		"log debug information to stderr",
	)
	command.PersistentFlags().BoolVar(
		&opts.Device,
		flagDevice,
		false,
		"use the device authorization grant instead of a browser",
	)
	command.PersistentFlags().BoolVar(
		&opts.Bearer,
		flagBearer,
		false,
		"return a Bearer credential on hosts that accept one (Git 2.46 or later)",
	)
}

// BindStorage registers the configure storage flag.
//
// Parameters:
//   - command: configure command.
//   - opts: destination for the storage value.
func BindStorage(command *cobra.Command, opts *Options) {
	command.Flags().StringVar(
		&opts.Storage,
		flagStorage,
		string(helper.StorageAuto),
		"credential storage helper ("+strings.Join(helper.Values(), "|")+")",
	)
}

// ParseStorage accepts the configure storage values.
//
// The accepted set comes from the helper package so the flag help and this
// validator cannot disagree about what is installable.
//
// Parameters:
//   - value: raw flag text.
//
// Returns:
//   - helper.Storage: accepted storage value.
//   - error: non-nil when value is not an accepted storage name.
func ParseStorage(value string) (helper.Storage, error) {
	if !helper.IsStorage(value) {
		return "", fmt.Errorf("%w: %q", errUnsupportedStorage, value)
	}

	return helper.Storage(value), nil
}
