// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package helper

const (
	// argConfig is the git subcommand that edits configuration.
	argConfig = "config"

	// argGlobal selects the per-user git config file.
	argGlobal = "--global"

	// argUnsetAll removes every matching config line.
	argUnsetAll = "--unset-all"

	// argFixedValue treats the value pattern as an exact string.
	//
	// Without it, an empty pattern is a regex that matches every helper.
	argFixedValue = "--fixed-value"

	// argAdd appends one value without replacing earlier helpers.
	argAdd = "--add"

	// keyCredentialHelper is the multi-valued credential helper key.
	keyCredentialHelper = "credential.helper"

	// helperOAuth is the git-credential-oauth helper without device flow.
	helperOAuth = "oauth"

	// helperOAuthDevice is the git-credential-oauth helper with device flow.
	//
	// Git appends the operation and runs the value as a command line, so the
	// flag has to be spelled the way this program parses it.
	helperOAuthDevice = "oauth --device"

	// helperOAuthDeviceSingleDash is the device helper as the upstream helper
	// writes it, and as this program wrote it before it was corrected. This
	// program does not accept the single dash, so the value is never installed.
	// It is only removed, so a leftover line cannot keep failing every request.
	helperOAuthDeviceSingleDash = "oauth -device"
)

// PlanConfigure returns git argument slices that install the helper.
//
// The slices omit the git binary. Order is unset-all, an empty-string
// reset, the storage helper unless it resolves to none, then oauth.
// Device flow uses the value "oauth --device".
//
// Parameters:
//   - opts: storage selection, device flow, and GOOS for auto.
//
// Returns:
//   - [][]string: git argument slices, in application order.
//   - error: non-nil when storage is unknown.
func PlanConfigure(opts Options) ([][]string, error) {
	helper, err := opts.Storage.resolve(opts.GOOS)
	if err != nil {
		return nil, err
	}

	plan := [][]string{
		unsetAllArgs(),
		addArgs(""),
	}

	if opts.Storage != StorageNone {
		plan = append(plan, addArgs(helper))
	}

	plan = append(plan, addArgs(oauthValue(opts)))

	return plan, nil
}

// PlanUnconfigure returns git argument slices that remove the helper.
//
// Each known value is removed with its own unset-all. The empty-string
// reset, the legacy cache helper, both oauth forms, and every storage
// helper are included. Values use --fixed-value so they are exact.
//
// Returns:
//   - [][]string: git argument slices, in removal order.
func PlanUnconfigure() [][]string {
	values := unconfigureValues()
	plan := make([][]string, 0, len(values))

	for _, value := range values {
		plan = append(plan, unsetValueArgs(value))
	}

	return plan
}

// oauthValue selects the oauth helper string.
//
// Parameters:
//   - opts: options whose device flag selects the helper form.
//
// Returns:
//   - string: "oauth" or "oauth --device".
func oauthValue(opts Options) string {
	if opts.Device {
		return helperOAuthDevice
	}

	return helperOAuth
}

// unconfigureValues lists credential.helper values to remove.
//
// The empty string is included. Cache is the legacy timeout form, not
// the bare cache name.
//
// Returns:
//   - []string: exact helper values, in removal order.
func unconfigureValues() []string {
	values := []string{"", helperCache}

	// StorageAuto installs whatever the platform picks and StorageNone installs
	// nothing, so neither can leave a value behind. StorageCache is already in
	// values under its legacy timeout form. The rest are removed by their own
	// name.
	for _, storage := range allStorages {
		switch storage {
		case StorageAuto, StorageNone, StorageCache:
			continue
		default:
			values = append(values, string(storage))
		}
	}

	return append(
		values,
		helperOAuth,
		helperOAuthDevice,
		helperOAuthDeviceSingleDash,
	)
}

// unsetAllArgs builds the command that clears every global helper.
//
// Returns:
//   - []string: git arguments, excluding the git binary.
func unsetAllArgs() []string {
	return []string{argConfig, argGlobal, argUnsetAll, keyCredentialHelper}
}

// addArgs builds the command that appends one helper value.
//
// Parameters:
//   - value: credential.helper value, which may be empty.
//
// Returns:
//   - []string: git arguments, excluding the git binary.
func addArgs(value string) []string {
	return []string{argConfig, argGlobal, argAdd, keyCredentialHelper, value}
}

// unsetValueArgs builds an exact unset of one helper value.
//
// --fixed-value is required. An empty pattern would otherwise match
// every credential.helper entry.
//
// Parameters:
//   - value: exact credential.helper value to remove.
//
// Returns:
//   - []string: git arguments, excluding the git binary.
func unsetValueArgs(value string) []string {
	return []string{
		argConfig,
		argGlobal,
		argUnsetAll,
		argFixedValue,
		keyCredentialHelper,
		value,
	}
}
