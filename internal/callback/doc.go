// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package callback receives the OAuth redirect on a loopback listener.
//
// RFC 8252 section 7.3 lets a native app claim an ephemeral port and listen
// only on the loopback interface, which is what this package does when no
// redirect URI is configured. The server is single-use: it takes the first
// request, hands the query to the caller, and shuts down.
package callback
