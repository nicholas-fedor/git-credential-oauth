// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package credential

import (
	"fmt"
	"io"
	"net/url"
	"slices"
	"strings"
)

// explicitFields records component keys present in the input.
type explicitFields struct {
	protocol bool
	host     bool
	path     bool
	username bool
	password bool
}

const (
	// spaceByte is the first byte value that may not appear on a protocol line.
	spaceByte = 0x20

	// deleteByte is the delete control code point.
	deleteByte = 0x7f

	// carriageReturn terminates a line sent with CRLF endings.
	carriageReturn = "\r"
)

// Parse reads a git credential request from reader.
//
// A read failure is wrapped and returned. An empty input is a zero request.
//
// Parameters:
//   - reader: source of protocol lines.
//
// Returns:
//   - Request: parsed attributes.
//   - error: wrapped read failure, or nil.
func Parse(reader io.Reader) (Request, error) {
	payload, err := io.ReadAll(reader)
	if err != nil {
		return emptyRequest(), fmt.Errorf("read credential request: %w", err)
	}

	return ParseString(string(payload))
}

// ParseString parses a git credential request from input.
//
// An empty input returns a zero request and a nil error.
//
// Parameters:
//   - input: protocol text.
//
// Returns:
//   - Request: parsed attributes.
//   - error: nil for string input, including empty input.
func ParseString(input string) (Request, error) {
	if input == "" {
		return emptyRequest(), nil
	}

	request := emptyRequest()
	seen := newExplicitFields()

	for line := range strings.SplitSeq(input, "\n") {
		applyLine(&request, &seen, strings.TrimRight(line, carriageReturn))
	}

	expandURL(&request, seen)

	return request, nil
}

// newExplicitFields returns component presence flags set to false.
//
// Returns:
//   - explicitFields: no component keys seen.
func newExplicitFields() explicitFields {
	return explicitFields{
		protocol: false,
		host:     false,
		path:     false,
		username: false,
		password: false,
	}
}

// applyLine records one protocol line.
//
// Parameters:
//   - request: request being filled.
//   - seen: component keys already present.
//   - line: one protocol line without a trailing carriage return.
func applyLine(request *Request, seen *explicitFields, line string) {
	key, value, ok := strings.Cut(line, "=")
	if !ok {
		return
	}

	if strings.HasSuffix(key, arraySuffix) {
		applyMulti(request, key, value)

		return
	}

	applyScalar(request, seen, key, value)
}

// applyMulti appends one multi-valued attribute.
//
// Parameters:
//   - request: request being filled.
//   - key: attribute name, including the [] suffix.
//   - value: attribute value, which may be empty.
func applyMulti(request *Request, key, value string) {
	switch key {
	case WWWAuth:
		request.WWWAuth = append(request.WWWAuth, value)
	case Capability:
		request.Capability = append(request.Capability, value)
	default:
		request.Extra = append(request.Extra, Pair{
			Key:   key,
			Value: value,
		})
	}
}

// applyScalar records one single-valued attribute.
//
// The last value wins. Unknown keys are stored in Extra.
//
// Parameters:
//   - request: request being filled.
//   - seen: component keys already present.
//   - key: attribute name.
//   - value: attribute value, which may be empty.
func applyScalar(request *Request, seen *explicitFields, key, value string) {
	switch key {
	case Protocol:
		request.Protocol = value
		seen.protocol = true

	case Host:
		request.Host = value
		seen.host = true

	case Path:
		request.Path = value
		seen.path = true

	case Username:
		request.Username = value
		seen.username = true

	case Password:
		request.Password = value
		seen.password = true

	case URL:
		request.URL = value

	case OAuthRefreshToken:
		request.OAuthRefreshToken = value

	default:
		upsertExtra(request, key, value)
	}
}

// upsertExtra stores the last value of an unknown single-valued key.
//
// Parameters:
//   - request: request being filled.
//   - key: attribute name.
//   - value: latest attribute value.
func upsertExtra(request *Request, key, value string) {
	for index := range request.Extra {
		if request.Extra[index].Key != key {
			continue
		}

		request.Extra[index].Value = value

		return
	}

	request.Extra = append(request.Extra, Pair{
		Key:   key,
		Value: value,
	})
}

// expandURL fills empty component fields from the url attribute.
//
// Expansion runs only when url is set and protocol or host is empty. Fields
// present in the input are left unchanged. A URL whose decoded components carry
// a control byte is not expanded at all: [url.Parse] percent-decodes, so a
// value such as url=%0A would otherwise inject a newline into the protocol
// transcript.
//
// Parameters:
//   - request: request being filled.
//   - seen: component keys already present.
func expandURL(request *Request, seen explicitFields) {
	if request.URL == "" {
		return
	}

	if request.Protocol != "" && request.Host != "" {
		return
	}

	parsed, err := url.Parse(request.URL)
	if err != nil {
		return
	}

	if !decodableComponents(parsed) {
		return
	}

	fillFromURL(request, seen, parsed)
}

// decodableComponents reports whether every component the URL yielded is free
// of control bytes.
//
// Scheme, host, and path are checked, as are the username and password when
// the URL carries user information.
//
// Parameters:
//   - parsed: parsed url attribute.
//
// Returns:
//   - bool: false when a decoded component carries a control byte.
func decodableComponents(parsed *url.URL) bool {
	components := []string{parsed.Scheme, parsed.Host, parsed.Path}

	if parsed.User != nil {
		components = append(
			components,
			usernameFromURL(parsed),
			passwordFromURL(parsed),
		)
	}

	return !slices.ContainsFunc(components, hasControlByte)
}

// hasControlByte reports whether value carries a C0 control byte or a delete.
//
// A control byte has no place on a credential protocol line. The url attribute
// is the only field whose bytes arrive percent-decoded, so this check closes
// the one route by which raw input could forge a transcript line.
//
// Parameters:
//   - value: decoded URL component.
//
// Returns:
//   - bool: true when a byte below space, or the delete byte, is present.
func hasControlByte(value string) bool {
	for index := range len(value) {
		if value[index] < spaceByte || value[index] == deleteByte {
			return true
		}
	}

	return false
}

// fillFromURL copies URL components into fields that were not set.
//
// Parameters:
//   - request: request being filled.
//   - seen: component keys already present.
//   - parsed: parsed url attribute.
func fillFromURL(request *Request, seen explicitFields, parsed *url.URL) {
	if !seen.protocol {
		request.Protocol = parsed.Scheme
	}

	if !seen.host {
		request.Host = parsed.Host
	}

	if !seen.username {
		request.Username = usernameFromURL(parsed)
	}

	if !seen.password {
		request.Password = passwordFromURL(parsed)
	}

	if !seen.path {
		request.Path = parsed.Path
	}
}

// usernameFromURL returns the username component of parsed.
//
// Parameters:
//   - parsed: parsed url attribute.
//
// Returns:
//   - string: username, or empty when the URL has none.
func usernameFromURL(parsed *url.URL) string {
	if parsed.User == nil {
		return ""
	}

	return parsed.User.Username()
}

// passwordFromURL returns the password component of parsed.
//
// Parameters:
//   - parsed: parsed url attribute.
//
// Returns:
//   - string: password, or empty when the URL has none.
func passwordFromURL(parsed *url.URL) string {
	if parsed.User == nil {
		return ""
	}

	password, ok := parsed.User.Password()
	if !ok {
		return ""
	}

	return password
}
