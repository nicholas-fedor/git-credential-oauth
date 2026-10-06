// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// Output holds the disjoint streams a credential helper writes to.
//
// Protocol carries the git credential wire format. Diag carries operator
// text, including device prompts and QR codes.
type Output struct {
	// In receives the credential request.
	In io.Reader

	// Protocol receives the credential response.
	Protocol io.Writer

	// Diag receives operator-facing diagnostics.
	Diag io.Writer
}

// errWriteDiagnostic is returned when a diagnostic write fails.
var errWriteDiagnostic = errors.New("write diagnostic")

// stdStreams returns the process standard streams.
//
// Returns:
//   - Output: stdin, stdout, and stderr.
func stdStreams() Output {
	return Output{
		In:       os.Stdin,
		Protocol: os.Stdout,
		Diag:     os.Stderr,
	}
}

// writeString writes text to writer.
//
// Parameters:
//   - writer: destination stream.
//   - text: exact bytes to write.
//
// Returns:
//   - error: wrapped write failure, or nil.
func writeString(writer io.Writer, text string) error {
	_, err := io.WriteString(writer, text)
	if err != nil {
		return fmt.Errorf("%w: %w", errWriteDiagnostic, err)
	}

	return nil
}

// writeDiagnostic writes text and drops a failure to write.
//
// The diagnostic stream is already the failure channel, so a write error
// cannot be reported further.
//
// Parameters:
//   - writer: destination stream.
//   - text: exact bytes to write.
func writeDiagnostic(writer io.Writer, text string) {
	err := writeString(writer, text)
	if err != nil {
		return
	}
}
