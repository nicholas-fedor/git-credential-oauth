// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package helper defines the configure and unconfigure commands, which install
// and remove this program as git's credential helper.
//
// Both commands are two halves of one operation, so they share a package and
// the Helper contract they call.
package helper
