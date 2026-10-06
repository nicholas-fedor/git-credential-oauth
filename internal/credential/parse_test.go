// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package credential

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// parseCase is one ParseString expectation.
type parseCase struct {
	name  string
	input string
	want  Request
}

// errReader returns a fixed error from Read.
type errReader struct {
	err error
}

// errWriter returns a fixed error from Write.
type errWriter struct {
	err error
}

// httpsProtocol is the https scheme used in parsing expectations.
const httpsProtocol = "https"

// noError fails the test when err is non-nil.
//
// Parameters:
//   - t: test handle.
//   - err: error to check.
func noError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

// same fails the test when want and got differ.
//
// Parameters:
//   - t: test handle.
//   - want: expected value.
//   - got: actual value.
func same(t *testing.T, want, got any) {
	t.Helper()

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// errorIs fails the test unless err matches target.
//
// Parameters:
//   - t: test handle.
//   - err: error to check.
//   - target: expected error.
func errorIs(t *testing.T, err, target error) {
	t.Helper()

	if !errors.Is(err, target) {
		t.Fatalf("got %v, want %v", err, target)
	}
}

// Read returns the configured error.
//
// Returns:
//   - int: zero bytes read.
//   - error: configured read failure.
func (reader errReader) Read([]byte) (int, error) {
	return 0, reader.err
}

// Write returns the configured error.
//
// Returns:
//   - int: zero bytes written.
//   - error: configured write failure.
func (writer errWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

// TestParseString covers protocol line parsing.
func TestParseString(t *testing.T) {
	t.Parallel()

	for _, tt := range parseCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseString(tt.input)
			noError(t, err)
			same(t, tt.want, got)
		})
	}
}

// TestParseStringEmpty returns a zero request and a nil error.
func TestParseStringEmpty(t *testing.T) {
	t.Parallel()

	got, err := ParseString("")
	noError(t, err)
	same(t, emptyRequest(), got)
}

// TestParseEmptyReader matches ParseString on empty input.
func TestParseEmptyReader(t *testing.T) {
	t.Parallel()

	got, err := Parse(strings.NewReader(""))
	noError(t, err)
	same(t, emptyRequest(), got)
}

// TestParseReaderError wraps the reader failure.
func TestParseReaderError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("read failed")
	_, err := Parse(errReader{err: sentinel})
	errorIs(t, err, sentinel)
}

// TestParseMatchesParseString agrees on carriage-return input.
func TestParseMatchesParseString(t *testing.T) {
	t.Parallel()

	const input = "protocol=https\r\nhost=example.com\r\n"

	fromString, err := ParseString(input)
	noError(t, err)

	fromReader, err := Parse(strings.NewReader(input))
	noError(t, err)
	same(t, fromString, fromReader)
}

// TestParseStringRoundTrip keeps typed fields stable after emit.
func TestParseStringRoundTrip(t *testing.T) {
	t.Parallel()

	const fixture = "capability[]=authtype\n" +
		"capability[]=state\n" +
		"protocol=https\n" +
		"host=example.com:8443\n" +
		"path=/repo.git\n" +
		"username=alice\n" +
		"password=secret\n" +
		"url=https://alice:secret@example.com:8443/repo.git\n" +
		"wwwauth[]=Basic\n" +
		"wwwauth[]=Bearer\n" +
		"oauth_refresh_token=refresh\n" +
		"state[]=one\n" +
		"state[]=two\n"

	first, err := ParseString(fixture)
	noError(t, err)

	second, err := ParseString(responseFromRequest(first).String())
	noError(t, err)
	same(t, first.Protocol, second.Protocol)
	same(t, first.Host, second.Host)
	same(t, first.Path, second.Path)
	same(t, first.Username, second.Username)
	same(t, first.Password, second.Password)
	same(t, first.URL, second.URL)
	same(t, first.WWWAuth, second.WWWAuth)
	same(t, first.Capability, second.Capability)
	same(t, first.OAuthRefreshToken, second.OAuthRefreshToken)
	same(t, first.Extra, second.Extra)
}

// parseCases returns the table for TestParseString.
//
// Returns:
//   - []parseCase: protocol parsing cases.
func parseCases() []parseCase {
	cases := scalarParseCases()
	cases = append(cases, repeatedParseCases()...)
	cases = append(cases, listParseCases()...)
	cases = append(cases, urlParseCases()...)

	return cases
}

// scalarParseCases returns single-value parsing cases.
//
// Returns:
//   - []parseCase: scalar protocol cases.
func scalarParseCases() []parseCase {
	return []parseCase{
		{
			name:  "missing equals",
			input: "protocol=https\nnot-a-pair\nhost=example.com\n",
			want: req(func(request *Request) {
				request.Protocol = httpsProtocol
				request.Host = "example.com"
			}),
		},
		{
			name:  "empty value",
			input: "username=\ncustom=\n",
			want: req(func(request *Request) {
				request.Username = ""
				request.Extra = []Pair{{
					Key:   "custom",
					Value: "",
				}}
			}),
		},
		{
			name:  "value containing equals",
			input: "password=a=b=c\n",
			want: req(func(request *Request) {
				request.Password = "a=b=c"
			}),
		},
		{
			name:  "carriage return",
			input: "protocol=https\r\nhost=example.com\r\n",
			want: req(func(request *Request) {
				request.Protocol = httpsProtocol
				request.Host = "example.com"
			}),
		},
		{
			name:  "repeated trailing carriage returns are all stripped",
			input: "protocol=https\r\r\r\nhost=example.com\r\r\r\n",
			want: req(func(request *Request) {
				request.Protocol = httpsProtocol
				request.Host = "example.com"
			}),
		},
		{
			name:  "value never keeps a trailing carriage return",
			input: "url=A:\r\r",
			want: req(func(request *Request) {
				request.Protocol = "a"
				request.URL = "A:"
			}),
		},
	}
}

// repeatedParseCases returns last-wins and refresh-token cases.
//
// Returns:
//   - []parseCase: repeated scalar cases.
func repeatedParseCases() []parseCase {
	return []parseCase{
		{
			name:  "repeated scalar key",
			input: "protocol=http\nprotocol=https\n",
			want: req(func(request *Request) {
				request.Protocol = httpsProtocol
			}),
		},
		{
			name:  "repeated extra key keeps last value",
			input: "custom=one\ncustom=two\n",
			want: req(func(request *Request) {
				request.Extra = []Pair{{
					Key:   "custom",
					Value: "two",
				}}
			}),
		},
		{
			name:  "oauth refresh token",
			input: "oauth_refresh_token=abc\n",
			want: req(func(request *Request) {
				request.OAuthRefreshToken = "abc"
			}),
		},
	}
}

// listParseCases returns multi-valued parsing cases.
//
// Returns:
//   - []parseCase: multi-valued protocol cases.
func listParseCases() []parseCase {
	return []parseCase{
		{
			name:  "repeated key list",
			input: "wwwauth[]=one\nwwwauth[]=two\n",
			want: req(func(request *Request) {
				request.WWWAuth = []string{"one", "two"}
			}),
		},
		{
			name:  "capability list",
			input: "capability[]=authtype\ncapability[]=state\n",
			want: req(func(request *Request) {
				request.Capability = []string{"authtype", "state"}
			}),
		},
		{
			name:  "other multi keys stay recoverable",
			input: "state[]=one\nstate[]=two\n",
			want: req(func(request *Request) {
				request.Extra = []Pair{
					{Key: "state[]", Value: "one"},
					{Key: "state[]", Value: "two"},
				}
			}),
		},
	}
}

// urlParseCases returns url shorthand cases.
//
// Returns:
//   - []parseCase: url expansion cases.
func urlParseCases() []parseCase {
	const shorthand = "https://alice:secret@example.com:8443/repo.git?query=1"

	return []parseCase{
		{
			name:  "url shorthand expansion",
			input: "url=" + shorthand + "\n",
			want: req(func(request *Request) {
				request.Protocol = httpsProtocol
				request.Host = "example.com:8443"
				request.Path = "/repo.git"
				request.Username = "alice"
				request.Password = "secret"
				request.URL = shorthand
			}),
		},
		{
			name:  "explicit protocol not overwritten by url",
			input: "protocol=ssh\nurl=https://example.com/repo.git\n",
			want: req(func(request *Request) {
				request.Protocol = "ssh"
				request.Host = "example.com"
				request.Path = "/repo.git"
				request.URL = "https://example.com/repo.git"
			}),
		},
		{
			name:  "explicit empty path not overwritten by url",
			input: "path=\nurl=https://example.com/repo.git\n",
			want: req(func(request *Request) {
				request.Protocol = httpsProtocol
				request.Host = "example.com"
				request.Path = ""
				request.URL = "https://example.com/repo.git"
			}),
		},
		{
			name: "url skipped when protocol and host are set",
			input: "protocol=https\nhost=keep.example\n" +
				"path=/keep.git\nurl=https://other.example/other.git\n",
			want: req(func(request *Request) {
				request.Protocol = httpsProtocol
				request.Host = "keep.example"
				request.Path = "/keep.git"
				request.URL = "https://other.example/other.git"
			}),
		},
		{
			name:  "percent-encoded newline in url path blocks expansion",
			input: "url=%0A\n",
			want: req(func(request *Request) {
				request.URL = "%0A"
			}),
		},
		{
			name:  "percent-encoded control byte in url path blocks expansion",
			input: "url=https://example.com/repo%0Agit\n",
			want: req(func(request *Request) {
				request.URL = "https://example.com/repo%0Agit"
			}),
		},
		{
			name:  "percent-encoded null in url path blocks expansion",
			input: "url=https://example.com/a%00b\n",
			want: req(func(request *Request) {
				request.URL = "https://example.com/a%00b"
			}),
		},
		{
			name:  "percent-encoded newline in url password blocks expansion",
			input: "url=https://user:se%0Acret@example.com\n",
			want: req(func(request *Request) {
				request.URL = "https://user:se%0Acret@example.com"
			}),
		},
		{
			name:  "percent-encoded carriage return in url host blocks expansion",
			input: "url=https://exa%0Dmple.com\n",
			want: req(func(request *Request) {
				request.URL = "https://exa%0Dmple.com"
			}),
		},
		{
			name:  "encoded space in url path still expands",
			input: "url=https://example.com/a%20b\n",
			want: req(func(request *Request) {
				request.Protocol = httpsProtocol
				request.Host = "example.com"
				request.Path = "/a b"
				request.URL = "https://example.com/a%20b"
			}),
		},
	}
}

// req builds a request from the zero value.
//
// Parameters:
//   - apply: mutator for the fields under test.
//
// Returns:
//   - Request: request with the mutated fields.
func req(apply func(*Request)) Request {
	request := emptyRequest()
	apply(&request)

	return request
}

// responseFromRequest emits logical fields in a stable order.
//
// Parameters:
//   - request: parsed request.
//
// Returns:
//   - *Response: attributes reconstructed from logical fields.
func responseFromRequest(request Request) *Response {
	response := NewResponse()
	addIfPresent(response, Protocol, request.Protocol)
	addIfPresent(response, Host, request.Host)
	addIfPresent(response, Path, request.Path)
	addIfPresent(response, Username, request.Username)
	addIfPresent(response, Password, request.Password)
	addIfPresent(response, URL, request.URL)
	addIfPresent(response, OAuthRefreshToken, request.OAuthRefreshToken)

	for _, value := range request.WWWAuth {
		response.Add(WWWAuth, value)
	}

	for _, value := range request.Capability {
		response.Add(Capability, value)
	}

	for _, pair := range request.Extra {
		response.Add(pair.Key, pair.Value)
	}

	return response
}

// addIfPresent appends a non-empty scalar attribute.
//
// Parameters:
//   - response: response being built.
//   - key: attribute name.
//   - value: attribute value.
func addIfPresent(response *Response, key, value string) {
	if value == "" {
		return
	}

	response.Add(key, value)
}
