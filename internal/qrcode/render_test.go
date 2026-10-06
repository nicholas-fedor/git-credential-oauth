// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package qrcode

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rsc.io/qr"
)

// errWriter returns a fixed error from Write.
type errWriter struct {
	err error
}

// Write returns the configured error without writing anything.
//
// Parameters:
//   - p: bytes the caller offered.
//
// Returns:
//   - int: zero bytes written.
//   - error: configured write failure.
func (w errWriter) Write(p []byte) (int, error) {
	return 0, w.err
}

// moduleCount counts the ANSI blocks in one rendered row.
//
// Each module is two escape sequences, one to set the background and one to
// reset it, so a block contributes two. The black and white constants also
// differ in byte length, which is why a row cannot be measured in bytes.
//
// Parameters:
//   - row: one rendered row.
//
// Returns:
//   - int: number of modules drawn in the row.
func moduleCount(row string) int {
	return strings.Count(row, "\033[") / 2
}

// TestRenderEmptyPayload returns an error and leaves the writer untouched.
func TestRenderEmptyPayload(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	err := Render(&buffer, "")
	require.ErrorIs(t, err, errEmptyData)
	assert.Empty(t, buffer.String())
}

// TestRenderShortPayload renders a short device-flow URL.
//
// A short payload is the device flow case, so it gets a stable code size that
// makes the rendered geometry assertable.
func TestRenderShortPayload(t *testing.T) {
	t.Parallel()

	const payload = "https://github.com/login/device"

	var buffer bytes.Buffer

	require.NoError(t, Render(&buffer, payload))

	code, err := qr.Encode(payload, qr.L)
	require.NoError(t, err)

	rows := strings.Split(strings.TrimSuffix(buffer.String(), newline), newline)

	// A quiet zone row, one row per module row, then a closing quiet zone row.
	require.Len(t, rows, code.Size+padding)

	// Every row draws the same number of modules. The two block constants are
	// different byte lengths, so modules are counted rather than bytes.
	wantModules := code.Size + padding

	for _, row := range rows {
		assert.Equal(t, wantModules, moduleCount(row), "modules per row")
	}
}

// TestRenderUsesBothBlockColors checks the finder pattern is actually drawn.
//
// A code that rendered every module white would still satisfy the geometry
// checks above, so the two colors are asserted to both appear.
func TestRenderUsesBothBlockColors(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	require.NoError(t, Render(&buffer, "https://github.com/login/device"))

	rendered := buffer.String()
	assert.Contains(t, rendered, blackBlock, "black modules")
	assert.Contains(t, rendered, whiteBlock, "white modules")
}

// TestRenderQuietZone checks the border rows are entirely white.
//
// The quiet zone keeps a scanner from reading an edge module as data, so the
// first and last rows must carry no black module.
func TestRenderQuietZone(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	require.NoError(t, Render(&buffer, "https://github.com/login/device"))

	rows := strings.Split(strings.TrimSuffix(buffer.String(), newline), newline)
	require.NotEmpty(t, rows)

	quiet := moduleCount(rows[0])
	assert.Zero(t, strings.Count(rows[0], blackBlock), "leading quiet zone")
	assert.Equal(t, quiet, moduleCount(rows[len(rows)-1]), "trailing quiet zone width")
	assert.Zero(t, strings.Count(rows[len(rows)-1], blackBlock), "trailing quiet zone")
}

// TestRenderWriteFailure surfaces a write failure.
func TestRenderWriteFailure(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")
	err := Render(errWriter{err: boom}, "https://github.com/login/device")

	require.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "write qr code")
}

// TestRenderFailsOnUnwritableWriter covers the render-to-buffer path failing.
//
// A bytes.Buffer never fails, so this asserts only the documented boundary:
// Render returns whatever the destination writer reports.
func TestRenderFailsOnUnwritableWriter(t *testing.T) {
	t.Parallel()

	var sink io.Writer = errWriter{err: errors.New("closed")}

	err := Render(sink, "data")
	require.Error(t, err)
}
