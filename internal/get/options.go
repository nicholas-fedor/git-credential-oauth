// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get

// Options are the per-invocation credential request settings.
//
// The zero value takes the authorization code grant and emits a password
// rather than a Bearer credential.
type Options struct {
	// UseBearer prefers HTTP Bearer authentication where the host supports it.
	//
	// The host must also advertise Bearer support and git must offer the
	// authtype capability, so this is a preference rather than a demand.
	UseBearer bool

	// Device requests the device authorization grant instead of the browser
	// flow. It is honored only where the provider supports the grant.
	Device bool
}
