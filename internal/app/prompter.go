// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/nicholas-fedor/git-credential-oauth/internal/qrcode"
)

// terminalPrompter writes device-flow instructions to the diagnostic stream.
type terminalPrompter struct {
	diag io.Writer
}

const (
	// deviceScanFormat is the prompt when a complete verification URI exists.
	deviceScanFormat = "Please scan the QR code or enter code %s at %s\n\n"

	// deviceCodeFormat is the prompt when only a verification URI exists.
	deviceCodeFormat = "Please enter code %s at %s\n"
)

// errRenderQR is returned when the device QR code cannot be rendered.
var errRenderQR = errors.New("render device qr code")

// errPromptCanceled is returned when the device prompt is canceled.
var errPromptCanceled = errors.New("device prompt canceled")

// PromptDevice writes device instructions and renders a QR code when possible.
//
// The instructions go to the diagnostic stream rather than the protocol stream,
// because git reads standard output as a credential response. A QR code is
// drawn only when the provider supplied a verification URI that already carries
// the user code; scanning that saves typing, while drawing a code from the bare
// verification URI would send the user to a page with nothing filled in.
//
// A render error is returned rather than swallowed, so a user who cannot see
// the code is told why instead of waiting for a device flow that cannot finish.
//
// Parameters:
//   - ctx: cancellation and deadline.
//   - userCode: code the user enters.
//   - verificationURI: URI where the user enters the code.
//   - verificationURIComplete: URI that already includes the code. Empty when
//     the provider did not supply one.
//
// Returns:
//   - error: non-nil when the context is done, the prompt fails, or the QR
//     render fails.
func (prompter terminalPrompter) PromptDevice(
	ctx context.Context,
	userCode string,
	verificationURI string,
	verificationURIComplete string,
) error {
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("%w: %w", errPromptCanceled, err)
	}

	if verificationURIComplete == "" {
		return writeString(
			prompter.diag,
			fmt.Sprintf(deviceCodeFormat, userCode, verificationURI),
		)
	}

	err = writeString(
		prompter.diag,
		fmt.Sprintf(deviceScanFormat, userCode, verificationURI),
	)
	if err != nil {
		return err
	}

	err = qrcode.Render(prompter.diag, verificationURIComplete)
	if err != nil {
		return fmt.Errorf("%w: %w", errRenderQR, err)
	}

	return nil
}
