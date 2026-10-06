// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import "context"

// Browser opens an authorization URL in a user agent.
type Browser interface {
	// Open opens targetURL.
	//
	// An error is returned to the caller. It is not treated as "no browser"
	// and it is not swallowed.
	//
	// Parameters:
	//   - ctx: cancellation and deadline.
	//   - targetURL: URL to open.
	//
	// Returns:
	//   - error: non-nil when the URL cannot be opened.
	Open(ctx context.Context, targetURL string) error
}
