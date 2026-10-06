// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMissingConfigErrorCarriesHint covers the operator-facing message.
//
// The hint is the only guidance a user gets, so it must survive to the error
// string rather than being logged and dropped.
func TestMissingConfigErrorCarriesHint(t *testing.T) {
	t.Parallel()

	withHint := &MissingConfigError{
		Host:   "gitlab.example.com",
		GitURL: "https://gitlab.example.com",
		Hint:   "Register an OAuth application for this host.",
	}
	assert.Equal(t, withHint.Hint, withHint.Error())
	require.ErrorIs(t, withHint, ErrMissingConfig)

	fallback := &MissingConfigError{Host: "gitlab.example.com"}
	assert.Contains(t, fallback.Error(), "gitlab.example.com")
	require.ErrorIs(t, fallback, ErrMissingConfig)
}

// TestNilMissingConfigErrorIsSafe covers a typed-nil receiver.
func TestNilMissingConfigErrorIsSafe(t *testing.T) {
	t.Parallel()

	var nilErr *MissingConfigError

	assert.Equal(t, ErrMissingConfig.Error(), nilErr.Error())
}

// TestMissingConfigErrorUnwrapsToTheSentinel checks the exit-status mapping.
//
// The program exits 2 for missing configuration by matching this sentinel, so
// the error has to unwrap to it through any wrapping the command adds.
func TestMissingConfigErrorUnwrapsToTheSentinel(t *testing.T) {
	t.Parallel()

	err := &MissingConfigError{Host: "git.example.com"}

	assert.Equal(t, ErrMissingConfig, err.Unwrap())
}
