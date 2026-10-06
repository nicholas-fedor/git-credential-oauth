// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errOutput = errors.New("output closed")

// TestStdStreamsKeepsTheProtocolOnStdout checks the stream split.
//
// Git reads the credential from standard output, so diagnostics must go to
// standard error or they would corrupt it.
func TestStdStreamsKeepsTheProtocolOnStdout(t *testing.T) {
	t.Parallel()

	out := stdStreams()

	assert.Equal(t, os.Stdin, out.In)
	assert.Equal(t, os.Stdout, out.Protocol)
	assert.Equal(t, os.Stderr, out.Diag)
}

// TestWriteStringWrapsAFailure covers both outcomes.
func TestWriteStringWrapsAFailure(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	require.NoError(t, writeString(&buffer, "hello"))
	assert.Equal(t, "hello", buffer.String())

	err := writeString(errDiagWriter{err: errOutput}, "hello")
	require.ErrorIs(t, err, errOutput)
	require.ErrorIs(t, err, errWriteDiagnostic)
}
