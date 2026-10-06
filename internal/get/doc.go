// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package get answers a git credential request with an OAuth token.
//
// This is the package that runs during a git operation. It reads the Git
// configuration for the requested URL, resolves it to an OAuth client, acquires
// a token, and emits the credential attributes in the order git expects. It
// never mutates Git configuration: that is the helper package's job.
package get
