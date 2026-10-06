// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import "context"

// Prompter displays a device-authorization prompt.
type Prompter interface {
	// PromptDevice shows the user code and verification URI.
	//
	// A non-nil error, including a QR render failure, aborts the grant.
	// It is not swallowed.
	//
	// Parameters:
	//   - ctx: cancellation and deadline.
	//   - userCode: code the user enters at the verification URI.
	//   - verificationURI: URI where the user enters userCode.
	//   - verificationURIComplete: URI that already includes the user code.
	//     Empty when the provider did not supply one.
	//
	// Returns:
	//   - error: non-nil when the prompt cannot be shown.
	PromptDevice(
		ctx context.Context,
		userCode, verificationURI, verificationURIComplete string,
	) error
}
