// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package helper

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValuesListsEveryStorageSorted checks the flag help and validation agree.
func TestValuesListsEveryStorageSorted(t *testing.T) {
	t.Parallel()

	values := Values()

	assert.True(t, slices.IsSorted(values))
	assert.Len(t, values, len(allStorages))

	for _, value := range values {
		assert.True(t, IsStorage(value), "value %q", value)
	}
}

// TestIsStorageIsExact checks near misses are refused.
func TestIsStorageIsExact(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "Libsecret", "keychain", "cache --timeout 21600", " store"} {
		assert.False(t, IsStorage(value), "value %q", value)
	}
}

// TestResolveWritesTheHelperValue covers every storage.
func TestResolveWritesTheHelperValue(t *testing.T) {
	t.Parallel()

	want := map[Storage]string{
		StorageLibsecret:   "libsecret",
		StorageOSXKeychain: "osxkeychain",
		StorageWincred:     "wincred",
		StorageCache:       "cache --timeout 21600",
		StorageStore:       "store",
		StorageNone:        "",
	}

	for storage, value := range want {
		got, err := storage.resolve("linux")
		require.NoError(t, err, "storage %q", storage)
		assert.Equal(t, value, got, "storage %q", storage)
	}

	_, err := Storage("keychain").resolve("darwin")
	require.ErrorIs(t, err, ErrUnknownStorage)
}

// TestAutoStorageFollowsThePlatform covers each platform the binary ships for.
func TestAutoStorageFollowsThePlatform(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "wincred", autoStorage("windows"))
	assert.Equal(t, "osxkeychain", autoStorage("darwin"))
	assert.Equal(t, "libsecret", autoStorage("linux"))
	assert.Equal(t, "libsecret", autoStorage("freebsd"))
}
