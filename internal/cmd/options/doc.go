// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package options holds the command inputs shared by every command package.
//
// The command packages under internal/cmd import this one, so it must not
// import them: the dependency runs from the root command down to this leaf.
package options
