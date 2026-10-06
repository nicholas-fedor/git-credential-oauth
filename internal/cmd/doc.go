// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package cmd wires the git-credential-oauth command tree.
//
// It parses flags and delegates to injected use cases. It does not look up
// executables, log, or terminate the process.
package cmd
