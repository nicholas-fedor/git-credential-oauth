// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package credential implements the git credential protocol wire format.
//
// Git writes key=value lines to a helper's standard input and reads the same
// form from its standard output, with a blank line ending each side. This
// package parses that request and renders a response, preserving attributes it
// does not model so nothing is silently dropped.
package credential
