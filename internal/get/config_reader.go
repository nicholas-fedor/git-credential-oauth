// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get

import "context"

// ConfigReader loads credential configuration without mutating git.
//
// A nil Deps.Config means every key is unset. A non-nil error from Get is
// returned to the caller and is never treated as an unset key.
type ConfigReader interface {
	// Get returns the git config value for key scoped to rawURL.
	//
	// Parameters:
	//   - ctx: cancellation and deadline.
	//   - key: full git config key, including the credential prefix.
	//   - rawURL: git URL used for urlmatch lookup.
	//
	// Returns:
	//   - value: configured value when found is true.
	//   - found: false when the key is unset.
	//   - err: non-nil when the lookup failed.
	Get(
		ctx context.Context,
		key, rawURL string,
	) (value string, found bool, err error)
}
