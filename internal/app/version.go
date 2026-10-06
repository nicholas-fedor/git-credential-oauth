// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import "github.com/nicholas-fedor/git-credential-oauth/internal/version"

// newVersion returns the version string the version command reports.
//
// GetVersion prefers the stamped value, so a dev build does not print the
// literal dev string when build info carries a module version.
//
// Returns:
//   - string: stamped version, or the module version for a dev build.
func newVersion() string {
	return version.GetVersion()
}
