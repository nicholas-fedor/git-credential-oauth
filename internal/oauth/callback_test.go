// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCloseCallback covers a nil, a clean, and a failing close.
func TestCloseCallback(t *testing.T) {
	t.Parallel()

	require.NoError(t, closeCallback(nil))

	clean := &stubCallback{}
	require.NoError(t, closeCallback(clean))
	assert.Equal(t, 1, clean.closed)

	err := closeCallback(&stubCallback{closeErr: errClose})
	require.ErrorIs(t, err, errClose)
	assert.Contains(t, err.Error(), "close callback")
}
