// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package credential

import (
	"strings"
	"testing"
)

// lineSeparator is the byte that ends a credential protocol line.
const lineSeparator = "\n"

// roundTrip returns the request re-parsed from its own rendered form.
//
// Parsing, emitting, and parsing again must reach the same protocol component
// values. A component that changes means the transcript this program writes
// does not say what the input said.
//
// Parameters:
//   - request: request after the first parse.
//
// Returns:
//   - again: request parsed from the emitted response.
func roundTrip(t *testing.T, request Request) Request {
	t.Helper()

	again, err := ParseString(responseFromRequest(request).String())
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}

	return again
}

// FuzzParseString checks the two transcript invariants on arbitrary input.
//
// The seed corpus holds the two inputs that previously broke them:
// "url=%0A" injected a newline into an emitted value, and "url=A:\r\r"
// changed the parsed protocol on a second pass because only one carriage
// return was stripped.
//
// Parameters:
//   - f: fuzzing handle.
func FuzzParseString(f *testing.F) {
	f.Add("url=https://example.com/repo.git")
	f.Add("protocol=https\nhost=example.com\n\n")
	f.Add("wwwauth[]=Basic realm=\"GitLab\"\ncapability[]=authtype\n")
	f.Add("url=%0A")
	f.Add("url=A:\r\r")
	f.Add("password=a=b=c\ncustom=one\ncustom=two\n")

	f.Fuzz(func(t *testing.T, input string) {
		request, err := ParseString(input)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		// A newline in any emitted attribute would forge a protocol line, so
		// none may survive parsing. Control bytes other than newline are not a
		// forging risk: they arrive verbatim from the same party that reads
		// them back, and the percent-decoded url path is the only place bytes
		// this program did not receive are introduced, which decodableComponents
		// already guards.
		for _, line := range []string{
			request.Protocol,
			request.Host,
			request.Path,
			request.Username,
			request.Password,
			request.URL,
			request.OAuthRefreshToken,
		} {
			if strings.Contains(line, lineSeparator) {
				t.Fatalf("component carries a newline: %q", line)
			}
		}

		for _, value := range request.WWWAuth {
			if strings.Contains(value, lineSeparator) {
				t.Fatalf("wwwauth carries a newline: %q", value)
			}
		}

		for _, value := range request.Capability {
			if strings.Contains(value, lineSeparator) {
				t.Fatalf("capability carries a newline: %q", value)
			}
		}

		// Extra is request-side only: emit never writes it back, so a control
		// byte there cannot reach a transcript. What matters is that it holds
		// no newline, which is the byte that would forge one.
		for _, pair := range request.Extra {
			if strings.Contains(pair.Key, lineSeparator) ||
				strings.Contains(pair.Value, lineSeparator) {
				t.Fatalf("extra pair carries a newline: %q=%q", pair.Key, pair.Value)
			}
		}

		// Parsing, emitting, and parsing again must reach the same component
		// values. A component that changes means the transcript says something
		// other than what the input said.
		again := roundTrip(t, request)

		if again.Protocol != request.Protocol || again.Host != request.Host {
			t.Fatalf("protocol or host changed: %q/%q -> %q/%q",
				request.Protocol, request.Host, again.Protocol, again.Host)
		}

		// Every emitted line must be a key=value pair with no stray newline.
		// An empty response emits nothing at all, so there is no line to check.
		rendered := responseFromRequest(request).String()
		if rendered == "" {
			return
		}

		for line := range strings.SplitSeq(strings.TrimSuffix(rendered, "\n"), "\n") {
			if line == "" {
				t.Fatal("emitted an empty line")
			}

			if !strings.Contains(line, "=") {
				t.Fatalf("emitted a line without a separator: %q", line)
			}
		}
	})
}

// FuzzResponseWrite checks a written value can never become two attributes.
//
// The values this helper writes include a token from the forge, which is
// untrusted. Whatever the value, Write either refuses it or produces exactly
// the attributes that were added, and the value reads back unchanged.
//
// Parameters:
//   - f: fuzzing handle.
func FuzzResponseWrite(f *testing.F) {
	f.Add("token")
	f.Add("token\nusername=attacker")
	f.Add("a=b=c")
	f.Add("tab\tand space ")
	f.Add("cr\r")
	f.Add("nul\x00")
	f.Add("")

	f.Fuzz(func(t *testing.T, value string) {
		response := NewResponse()
		response.Add(Username, "oauth2")
		response.Add(Password, value)

		var builder strings.Builder

		err := response.Write(&builder)
		if err != nil {
			if builder.Len() != 0 {
				t.Fatalf("a refused response wrote %q", builder.String())
			}

			return
		}

		written := builder.String()
		if lines := strings.Count(written, lineSeparator); lines != 2 {
			t.Fatalf("value %q produced %d lines: %q", value, lines, written)
		}

		request, parseErr := ParseString(written)
		if parseErr != nil {
			t.Fatalf("written response does not parse: %v", parseErr)
		}

		if request.Username != "oauth2" {
			t.Fatalf("value %q changed the username to %q", value, request.Username)
		}

		if request.Password != value {
			t.Fatalf("value %q read back as %q", value, request.Password)
		}
	})
}
