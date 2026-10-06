// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package qrcode renders a device-flow verification URI as a QR code.
//
// Git has no display, so the device grant writes the URI to standard error
// and draws a scannable code next to it. Each QR module is drawn as two
// spaces on a black or white background, with a one-module quiet zone so a
// scanner does not read an edge module as data.
package qrcode
