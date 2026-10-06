// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package browser starts the platform web browser.
//
// The browser is only needed for the authorization code grant: the device
// grant prints a URL for the user to open, and a refresh needs no browser at
// all. Opening is delegated to a third-party command, so this package holds no
// browser state.
package browser
