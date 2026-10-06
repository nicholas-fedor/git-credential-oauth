// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package git runs the git binary and reports typed failures.
//
// Only two things are needed from git: read a URL-matched configuration value,
// and run a configuration subcommand. Every failure becomes an *Error carrying
// the exit code and stderr, so a missing key is never confused with a broken
// git. The credential section key names live here as well.
package git
