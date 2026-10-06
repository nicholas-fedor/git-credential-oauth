// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package qrcode

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"rsc.io/qr"
)

const (
	// blackBlock is one black QR module, two spaces on a black background.
	blackBlock = "\033[40m  \033[0m"
	// whiteBlock is one white QR module, two spaces on a white background.
	whiteBlock = "\033[107m  \033[0m"
	// padding is the extra module count for a one-module quiet zone per side.
	padding = 2
	// newline separates rendered rows.
	newline = "\n"
)

// errEmptyData is returned when Render is given an empty payload.
var errEmptyData = errors.New("empty qr payload")

// Render writes data as an ANSI block QR code to w.
//
// The code uses low error correction. Each module is two spaces on a black
// or white background, with a one-module quiet zone on every side.
// An empty payload or an encode failure writes nothing.
//
// Parameters:
//   - w: destination for the rendered code.
//   - data: text to encode.
//
// Returns:
//   - error: empty payload, encode failure, or write failure.
func Render(w io.Writer, data string) error {
	writer := w

	// Reject empty input before encoding so the writer stays untouched.
	if data == "" {
		return errEmptyData
	}

	code, err := qr.Encode(data, qr.L)
	if err != nil {
		return fmt.Errorf("encode qr code: %w", err)
	}

	var buffer bytes.Buffer

	err = renderInto(&buffer, code)
	if err != nil {
		return err
	}

	// Flush only after a successful encode. Surface a short write.
	_, err = buffer.WriteTo(writer)
	if err != nil {
		return fmt.Errorf("write qr code: %w", err)
	}

	return nil
}

// renderInto appends the ANSI rendering of code to buffer.
//
// A quiet-zone border is written before and after the modules.
//
// Parameters:
//   - buffer: destination for the ANSI text.
//   - code: QR code to render.
//
// Returns:
//   - error: buffer write failure.
func renderInto(buffer *bytes.Buffer, code *qr.Code) error {
	border := strings.Repeat(whiteBlock, code.Size+padding) + newline

	err := appendBlock(buffer, border)
	if err != nil {
		return err
	}

	for row := range code.Size {
		err := appendRow(buffer, code, row)
		if err != nil {
			return err
		}
	}

	return appendBlock(buffer, border)
}

// appendRow writes one module row plus its quiet-zone columns.
//
// Parameters:
//   - buffer: destination for the ANSI text.
//   - code: QR code being rendered.
//   - row: zero-based module row.
//
// Returns:
//   - error: buffer write failure.
func appendRow(buffer *bytes.Buffer, code *qr.Code, row int) error {
	err := appendBlock(buffer, whiteBlock)
	if err != nil {
		return err
	}

	for column := range code.Size {
		err = appendBlock(buffer, moduleBlock(code, column, row))
		if err != nil {
			return err
		}
	}

	err = appendBlock(buffer, whiteBlock)
	if err != nil {
		return err
	}

	return appendBlock(buffer, newline)
}

// moduleBlock returns the ANSI block for the module at column, row.
//
// Parameters:
//   - code: QR code being rendered.
//   - column: zero-based module column.
//   - row: zero-based module row.
//
// Returns:
//   - string: black or white ANSI block.
func moduleBlock(code *qr.Code, column, row int) string {
	if code.Black(column, row) {
		return blackBlock
	}

	return whiteBlock
}

// appendBlock writes value to buffer.
//
// Parameters:
//   - buffer: destination for the ANSI text.
//   - value: text to append.
//
// Returns:
//   - error: buffer write failure.
func appendBlock(buffer *bytes.Buffer, value string) error {
	_, err := buffer.WriteString(value)
	if err != nil {
		return fmt.Errorf("buffer qr block: %w", err)
	}

	return nil
}
