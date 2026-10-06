// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"fmt"

	"golang.org/x/oauth2"
)

// Nonce returns a new CSRF state value.
//
// Returns:
//   - string: non-empty state.
//   - error: non-nil when the nonce cannot be generated.
type Nonce func() (string, error)

// newState returns a CSRF state from nonce, or a PKCE verifier when nonce is
// nil.
//
// Parameters:
//   - nonce: state generator. Nil uses oauth2.GenerateVerifier.
//
// Returns:
//   - string: CSRF state.
//   - error: nonce failure.
func newState(nonce Nonce) (string, error) {
	if nonce == nil {
		return oauth2.GenerateVerifier(), nil
	}

	state, err := nonce()
	if err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}

	if state == "" {
		return "", errEmptyState
	}

	return state, nil
}
