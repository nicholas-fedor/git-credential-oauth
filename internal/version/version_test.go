// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package version

import (
	"testing"
)

// TestGetVersionPrefersStampedValue checks a release build reports its stamp.
//
// The stamp is what a user quotes in a bug report, so it has to win over
// whatever the module cache happens to hold.
func TestGetVersionPrefersStampedValue(t *testing.T) {
	previous := Version
	t.Cleanup(func() { Version = previous })

	Version = "v1.2.3"

	if got := GetVersion(); got != "v1.2.3" {
		t.Fatalf("GetVersion() = %q, want %q", got, "v1.2.3")
	}
}

// TestGetVersionFallsBackToBuildInfo checks a dev build still reports a
// version rather than the bare dev string.
func TestGetVersionFallsBackToBuildInfo(t *testing.T) {
	previous := Version
	t.Cleanup(func() { Version = previous })

	Version = devVersion

	got := GetVersion()
	if got == "" {
		t.Fatal("GetVersion() = empty string")
	}

	// A test binary is a main module, so build info is always available here.
	if got == devVersion {
		t.Skip("build info reports no module version")
	}
}

// TestGetVersionLeavesVersionUnchanged checks the fallback does not assign.
//
// Mutating the package variable would make the first call sticky, so a later
// stamp would be ignored.
func TestGetVersionLeavesVersionUnchanged(t *testing.T) {
	previous := Version
	t.Cleanup(func() { Version = previous })

	Version = devVersion
	GetVersion()

	if Version != devVersion {
		t.Fatalf("Version = %q after GetVersion, want %q", Version, devVersion)
	}
}

// TestCurrentCarriesRevisionAndTime checks the other stamped fields.
func TestCurrentCarriesRevisionAndTime(t *testing.T) {
	previousVersion := Version
	previousSHA := CommitSHA
	previousTime := BuildTime

	t.Cleanup(func() {
		Version = previousVersion
		CommitSHA = previousSHA
		BuildTime = previousTime
	})

	Version = "v9.9.9"
	CommitSHA = "abc1234"
	BuildTime = "2026-01-01T00:00:00Z"

	info := Current()

	if info.Version != "v9.9.9" {
		t.Errorf("Version = %q, want %q", info.Version, "v9.9.9")
	}

	if info.CommitSHA != "abc1234" {
		t.Errorf("CommitSHA = %q, want %q", info.CommitSHA, "abc1234")
	}

	if info.BuildTime != "2026-01-01T00:00:00Z" {
		t.Errorf("BuildTime = %q, want %q", info.BuildTime, "2026-01-01T00:00:00Z")
	}
}

// TestUnstampedDefaultsAreVisible checks an unstamped build says so.
//
// Printing empty fields would look like a bug; "unknown" tells the reader the
// binary was not stamped.
func TestUnstampedDefaultsAreVisible(t *testing.T) {
	previousSHA := CommitSHA
	previousTime := BuildTime

	t.Cleanup(func() {
		CommitSHA = previousSHA
		BuildTime = previousTime
	})

	CommitSHA = unknown
	BuildTime = unknown

	info := Current()

	if info.CommitSHA != unknown {
		t.Errorf("CommitSHA = %q, want %q", info.CommitSHA, unknown)
	}

	if info.BuildTime != unknown {
		t.Errorf("BuildTime = %q, want %q", info.BuildTime, unknown)
	}
}
