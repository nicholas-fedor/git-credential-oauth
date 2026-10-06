// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package helper

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlanConfigure covers storage selection, device flow, and reset order.
func TestPlanConfigure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    Options
		want    []string
		wantErr bool
	}{
		{
			name: "auto on linux selects libsecret",
			opts: Options{Storage: StorageAuto, GOOS: "linux"},
			want: []string{"", "libsecret", "oauth"},
		},
		{
			name: "auto on darwin selects osxkeychain",
			opts: Options{Storage: StorageAuto, GOOS: "darwin"},
			want: []string{"", "osxkeychain", "oauth"},
		},
		{
			name: "auto on windows selects wincred",
			opts: Options{Storage: StorageAuto, GOOS: "windows"},
			want: []string{"", "wincred", "oauth"},
		},
		{
			name: "auto on an unknown platform selects libsecret",
			opts: Options{Storage: StorageAuto, GOOS: "plan9"},
			want: []string{"", "libsecret", "oauth"},
		},
		{
			name: "explicit storage ignores goos",
			opts: Options{Storage: StorageOSXKeychain, GOOS: "windows"},
			want: []string{"", "osxkeychain", "oauth"},
		},
		{
			name: "cache carries its timeout",
			opts: Options{Storage: StorageCache, GOOS: "linux"},
			want: []string{"", "cache --timeout 21600", "oauth"},
		},
		{
			name: "none omits the storage helper",
			opts: Options{Storage: StorageNone, GOOS: "linux"},
			want: []string{"", "oauth"},
		},
		{
			name: "device flow uses the device helper value",
			opts: Options{Storage: StorageNone, GOOS: "linux", Device: true},
			want: []string{"", "oauth --device"},
		},
		{
			name:    "unknown storage is rejected",
			opts:    Options{Storage: "keychain-ish", GOOS: "linux"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan, err := PlanConfigure(tt.opts)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrUnknownStorage)
				assert.Empty(t, plan)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, addedValues(plan), "added helper values")
		})
	}
}

// TestPlanConfigureCommandOrder checks the reset precedes every add.
//
// An empty-string add resets the helper list, so it has to run before the
// storage and oauth values are appended.
func TestPlanConfigureCommandOrder(t *testing.T) {
	t.Parallel()

	plan, err := PlanConfigure(Options{Storage: StorageStore, GOOS: "linux"})
	require.NoError(t, err)

	require.Len(t, plan, 4)
	assert.Equal(t, []string{"config", "--global", "--unset-all", "credential.helper"}, plan[0])
	assert.Equal(t, []string{"config", "--global", "--add", "credential.helper", ""}, plan[1])
	assert.Equal(t, []string{"config", "--global", "--add", "credential.helper", "store"}, plan[2])
	assert.Equal(t, []string{"config", "--global", "--add", "credential.helper", "oauth"}, plan[3])
}

// TestPlanUnconfigureRemovesEverythingConfigureInstalls checks symmetry.
func TestPlanUnconfigureRemovesEverythingConfigureInstalls(t *testing.T) {
	t.Parallel()

	storages := []Storage{
		StorageAuto,
		StorageLibsecret,
		StorageOSXKeychain,
		StorageWincred,
		StorageCache,
		StorageStore,
		StorageNone,
	}
	gooses := []string{"linux", "darwin", "windows"}
	devices := []bool{false, true}

	plan := PlanUnconfigure()
	removed := removedValues(plan)

	for _, storage := range storages {
		for _, goos := range gooses {
			for _, device := range devices {
				opts := Options{Storage: storage, GOOS: goos, Device: device}

				installed, err := PlanConfigure(opts)
				require.NoError(t, err)

				for _, value := range addedValues(installed) {
					assert.Contains(t, removed, value,
						"unconfigure must remove %q for %+v", value, opts)
				}
			}
		}
	}
}

// TestPlanUnconfigureUsesFixedValue checks the exact-match flag is present.
//
// Without --fixed-value an empty pattern is a regular expression that matches
// every helper, so the unset would remove unrelated entries.
func TestPlanUnconfigureUsesFixedValue(t *testing.T) {
	t.Parallel()

	for _, args := range PlanUnconfigure() {
		assert.Equal(t, []string{
			"config", "--global", "--unset-all",
			"--fixed-value", "credential.helper", args[len(args)-1],
		}, args)
	}
}

// TestPlanUnconfigureCoversEmptyAndLegacyCache checks the two special values.
func TestPlanUnconfigureCoversEmptyAndLegacyCache(t *testing.T) {
	t.Parallel()

	removed := removedValues(PlanUnconfigure())

	assert.Contains(t, removed, "", "empty-string reset")
	assert.Contains(t, removed, "cache --timeout 21600", "legacy cache timeout form")
	assert.NotContains(t, removed, "cache", "bare cache is not the legacy form")
}

// TestPlanUnconfigureRemovesTheSingleDashDeviceForm checks a leftover line goes.
//
// The upstream helper writes "oauth -device", and so did this program before
// the flag spelling was corrected. This program rejects that line on every
// request, so unconfigure has to remove it even though configure never writes
// it.
func TestPlanUnconfigureRemovesTheSingleDashDeviceForm(t *testing.T) {
	t.Parallel()

	removed := removedValues(PlanUnconfigure())

	assert.Contains(t, removed, "oauth --device", "the form configure writes")
	assert.Contains(t, removed, "oauth -device", "the form it used to write")

	for _, storage := range []Storage{StorageNone, StorageCache} {
		installed, err := PlanConfigure(Options{Storage: storage, GOOS: "linux", Device: true})
		require.NoError(t, err)
		assert.NotContains(t, addedValues(installed), "oauth -device",
			"configure must not install a line this program rejects")
	}
}

// addedValues returns the value argument of every add command in a plan.
//
// Parameters:
//   - plan: git argument slices.
//
// Returns:
//   - []string: helper values in application order.
func addedValues(plan [][]string) []string {
	values := make([]string, 0, len(plan))

	for _, args := range plan {
		if !slices.Contains(args, "--add") {
			continue
		}

		values = append(values, args[len(args)-1])
	}

	return values
}

// removedValues returns the value argument of every exact unset in a plan.
//
// Parameters:
//   - plan: git argument slices.
//
// Returns:
//   - []string: helper values in removal order.
func removedValues(plan [][]string) []string {
	values := make([]string, 0, len(plan))

	for _, args := range plan {
		if !slices.Contains(args, "--fixed-value") {
			continue
		}

		values = append(values, args[len(args)-1])
	}

	return values
}
