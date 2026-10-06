// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errDiagWriter returns a fixed error from Write.
type errDiagWriter struct {
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
func (w errDiagWriter) Write(p []byte) (int, error) {
	return 0, w.err
}

// TestPromptDeviceWithCompleteURI renders a QR code.
//
// A provider that supplies a verification_uri_complete lets the user scan
// instead of typing, which is the better path and must be taken when offered.
func TestPromptDeviceWithCompleteURI(t *testing.T) {
	t.Parallel()

	var diag bytes.Buffer

	prompter := terminalPrompter{diag: &diag}
	err := prompter.PromptDevice(
		t.Context(),
		"ABCD-EFGH",
		"https://example.com/device",
		"https://example.com/device?code=ABCD-EFGH",
	)

	require.NoError(t, err)
	rendered := diag.String()

	assert.Contains(t, rendered, "ABCD-EFGH", "user code")
	assert.Contains(t, rendered, "https://example.com/device", "verification URI")
	assert.Contains(t, rendered, "\033[", "QR escape sequence")
}

// TestPromptDeviceWithoutCompleteURI prints the code only.
//
// Not every provider supplies a complete URI, and rendering a QR code from the
// bare verification URI would send the user to a page with no code in it.
func TestPromptDeviceWithoutCompleteURI(t *testing.T) {
	t.Parallel()

	var diag bytes.Buffer

	prompter := terminalPrompter{diag: &diag}
	err := prompter.PromptDevice(t.Context(), "ABCD-EFGH", "https://example.com/device", "")

	require.NoError(t, err)
	rendered := diag.String()

	assert.Contains(t, rendered, "ABCD-EFGH")
	assert.Contains(t, rendered, "https://example.com/device")
	assert.NotContains(t, rendered, "\033[", "no QR code without a complete URI")
}

// TestPromptDeviceUsesCompleteURIIsNotShown checks the complete URI is not
// printed as plain text.
//
// It already encodes the user code, so printing it too would put the code on
// screen twice for no reason.
func TestPromptDeviceDoesNotPrintCompleteURI(t *testing.T) {
	t.Parallel()

	var diag bytes.Buffer

	const complete = "https://example.com/device?code=SECRET"

	prompter := terminalPrompter{diag: &diag}
	require.NoError(t, prompter.PromptDevice(
		t.Context(),
		"ABCD-EFGH",
		"https://example.com/device",
		complete,
	))

	assert.NotContains(t, diag.String(), complete)
}

// TestPromptDeviceCanceled checks a canceled context stops the prompt.
func TestPromptDeviceCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var diag bytes.Buffer

	prompter := terminalPrompter{diag: &diag}
	err := prompter.PromptDevice(ctx, "CODE", "https://example.com", "")

	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, err, errPromptCanceled)
	assert.Empty(t, diag.String(), "nothing written after cancel")
}

// TestPromptDeviceWriteFailure surfaces a diagnostic stream failure.
func TestPromptDeviceWriteFailure(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")
	prompter := terminalPrompter{diag: errDiagWriter{err: boom}}

	err := prompter.PromptDevice(t.Context(), "CODE", "https://example.com", "")
	require.ErrorIs(t, err, boom)
}
