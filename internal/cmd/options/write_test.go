// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package options

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteString checks the success and failure paths.
//
// A command that cannot report its own failure to git turns a write error into
// a silent credential failure, so the error must be wrapped and returned.
func TestWriteString(t *testing.T) {
	t.Parallel()

	t.Run("writes the exact bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		require.NoError(t, WriteString(&buf, "value 0\n"))
		assert.Equal(t, "value 0\n", buf.String())
	})

	t.Run("wraps a write failure", func(t *testing.T) {
		t.Parallel()

		err := WriteString(failWriter{}, "value 0\n")
		require.ErrorIs(t, err, ErrWriteOutput)
	})
}
