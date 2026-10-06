// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package credential

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

// Response is an ordered git credential protocol reply.
type Response struct {
	pairs []Pair
}

// errNilResponse is returned when Write is called on a nil response.
var (
	errNilResponse = errors.New("nil credential response")

	// ErrUnsafeAttribute is returned when a key or value could not be written
	// without changing the meaning of the stream.
	//
	// A newline ends an attribute, so a value containing one would add
	// attributes of its own: a token from a hostile server could set the
	// username or the expiry Git stores. Git refuses to write such a value for
	// the same reason. A carriage return, which readers strip from the end of a
	// line, a NUL byte, an empty key, and a key containing "=" are refused too,
	// since none can be read back as written.
	ErrUnsafeAttribute = errors.New("unsafe credential attribute")
)

// NewResponse returns an empty credential response.
//
// Returns:
//   - *Response: response that preserves Add order.
func NewResponse() *Response {
	return &Response{
		pairs: nil,
	}
}

// Add appends one attribute in call order.
//
// Parameters:
//   - key: attribute name.
//   - value: attribute value.
func (resp *Response) Add(key, value string) {
	resp.pairs = append(resp.pairs, Pair{
		Key:   key,
		Value: value,
	})
}

// Pairs returns the attributes in Add order.
//
// The returned slice is a copy, so reordering it does not change Write.
//
// Returns:
//   - []Pair: attributes in call order.
func (resp *Response) Pairs() []Pair {
	return slices.Clone(resp.pairs)
}

// String returns the protocol text that Write emits.
//
// Returns:
//   - string: key=value lines in Add order.
func (resp *Response) String() string {
	return formatPairs(resp.pairs)
}

// Write emits key=value lines in Add order.
//
// A nil response returns an error. The output matches String. An attribute
// that could not be read back as written is refused before anything is
// written, so Git never sees part of a response.
//
// Parameters:
//   - writer: destination for protocol lines.
//
// Returns:
//   - error: nil response, ErrUnsafeAttribute, or a wrapped write failure.
func (resp *Response) Write(writer io.Writer) error {
	if resp == nil {
		return errNilResponse
	}

	err := validatePairs(resp.pairs)
	if err != nil {
		return err
	}

	_, err = io.WriteString(writer, resp.String())
	if err != nil {
		return fmt.Errorf("write credential response: %w", err)
	}

	return nil
}

// formatPairs renders attributes as key=value lines.
//
// Parameters:
//   - pairs: attributes in emit order.
//
// Returns:
//   - string: protocol text, or empty when pairs is empty.
func formatPairs(pairs []Pair) string {
	if len(pairs) == 0 {
		return ""
	}

	lines := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		lines = append(lines, pair.Key+"="+pair.Value)
	}

	return strings.Join(lines, "\n") + "\n"
}

// validatePairs checks every attribute can be written and read back unchanged.
//
// Parameters:
//   - pairs: attributes in write order.
//
// Returns:
//   - error: ErrUnsafeAttribute naming the first key that cannot be written.
func validatePairs(pairs []Pair) error {
	for _, pair := range pairs {
		if pair.Key == "" || strings.ContainsAny(pair.Key, "=\r\n\x00") {
			return fmt.Errorf("%w: key %q", ErrUnsafeAttribute, pair.Key)
		}

		if strings.ContainsAny(pair.Value, "\r\n\x00") {
			return fmt.Errorf("%w: value for %s", ErrUnsafeAttribute, pair.Key)
		}
	}

	return nil
}
