// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewStateDefaultsToARandomValue checks the default source.
//
// The state is what ties a redirect to this sign-in, so the default must be
// unpredictable and different every time.
func TestNewStateDefaultsToARandomValue(t *testing.T) {
	t.Parallel()

	first, err := newState(nil)
	require.NoError(t, err)

	second, err := newState(nil)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(first), 43, "at least 256 bits, base64url encoded")
	assert.NotEqual(t, first, second)
}

// TestNewStateUsesTheInjectedSource covers a custom source.
func TestNewStateUsesTheInjectedSource(t *testing.T) {
	t.Parallel()

	state, err := newState(func() (string, error) { return "fixed", nil })
	require.NoError(t, err)
	assert.Equal(t, "fixed", state)
}

// TestNewStateRejectsAFailingOrEmptySource checks a weak state is never used.
//
// An empty state would match a redirect that carries none, which defeats the
// check entirely.
func TestNewStateRejectsAFailingOrEmptySource(t *testing.T) {
	t.Parallel()

	_, err := newState(func() (string, error) { return "", errStub })
	require.ErrorIs(t, err, errStub)

	_, err = newState(func() (string, error) { return "", nil })
	require.ErrorIs(t, err, errEmptyState)
}
