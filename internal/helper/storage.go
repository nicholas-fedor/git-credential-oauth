// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package helper

import (
	"fmt"
	"slices"
)

// Storage names the credential helper installed ahead of oauth.
type Storage string

const (
	// StorageAuto selects the platform keyring from GOOS.
	StorageAuto Storage = "auto"

	// StorageLibsecret stores credentials with the libsecret helper.
	StorageLibsecret Storage = "libsecret"

	// StorageOSXKeychain stores credentials in the macOS keychain.
	StorageOSXKeychain Storage = "osxkeychain"

	// StorageWincred stores credentials in the Windows credential manager.
	StorageWincred Storage = "wincred"

	// StorageCache stores credentials in a short-lived memory cache.
	StorageCache Storage = "cache"

	// StorageStore stores credentials in a plaintext file.
	StorageStore Storage = "store"

	// StorageNone omits the storage helper and keeps only the oauth helper.
	StorageNone Storage = "none"

	// helperCache is the cache helper value, including its timeout.
	helperCache = "cache --timeout 21600"

	// goosWindows is the GOOS name that selects wincred for auto.
	goosWindows = "windows"

	// goosDarwin is the GOOS name that selects osxkeychain for auto.
	goosDarwin = "darwin"
)

// allStorages lists every accepted storage value in help-text order.
//
// The flag help, the validator, and the unconfigure plan all read this list, so
// a new backend is one entry rather than four coordinated edits.
//
// StorageAuto is a request rather than a helper, and StorageNone installs
// nothing, so neither appears in the unconfigure plan.
var allStorages = []Storage{
	StorageAuto,
	StorageLibsecret,
	StorageOSXKeychain,
	StorageWincred,
	StorageCache,
	StorageStore,
	StorageNone,
}

// Values returns every accepted storage name, sorted for the flag help.
//
// Returns:
//   - []string: accepted names, safe to mutate.
func Values() []string {
	names := make([]string, 0, len(allStorages))
	for _, storage := range allStorages {
		names = append(names, string(storage))
	}

	slices.Sort(names)

	return names
}

// IsStorage reports whether value names an accepted storage.
//
// Parameters:
//   - value: raw flag text.
//
// Returns:
//   - bool: true for a known storage name.
func IsStorage(value string) bool {
	return slices.Contains(allStorages, Storage(value))
}

// resolve maps storage to the helper value git should install.
//
// StorageNone returns an empty value and a nil error. The caller omits that
// helper. Explicit helpers are returned unchanged, ignoring GOOS.
//
// Parameters:
//   - goos: operating system name used only when storage is auto.
//
// Returns:
//   - string: helper value, or empty when the storage helper is omitted.
//   - error: non-nil when storage is not a known value.
func (storage Storage) resolve(goos string) (string, error) {
	if !IsStorage(string(storage)) {
		return "", fmt.Errorf("credential storage %q: %w", storage, ErrUnknownStorage)
	}

	if storage == StorageAuto {
		return autoStorage(goos), nil
	}

	if storage == StorageCache {
		return helperCache, nil
	}

	if storage == StorageNone {
		return "", nil
	}

	return string(storage), nil
}

// autoStorage selects the platform keyring for StorageAuto.
//
// Empty and unrecognized GOOS values use libsecret.
//
// Parameters:
//   - goos: operating system name, such as linux, darwin, or windows.
//
// Returns:
//   - string: wincred, osxkeychain, or libsecret.
func autoStorage(goos string) string {
	switch goos {
	case goosWindows:
		return string(StorageWincred)
	case goosDarwin:
		return string(StorageOSXKeychain)
	default:
		// Linux, empty, and any other GOOS use the libsecret helper.
		return string(StorageLibsecret)
	}
}
