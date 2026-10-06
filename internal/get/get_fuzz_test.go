// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get

import (
	"net/url"
	"testing"
)

// FuzzResolveReference checks a configured endpoint can never be cleartext.
//
// The resolved URL receives the authorization code and the token. Whatever the
// operator's config or the remote contains, an accepted result must parse and
// use https.
//
// Parameters:
//   - f: fuzzing handle.
func FuzzResolveReference(f *testing.F) {
	f.Add("https://git.example.com", "/oauth/token")
	f.Add("https://git.example.com", "oauth/token")
	f.Add("https://git.example.com", "//sso.example.com/token")
	f.Add("https://git.example.com", "http://evil.example.com/token")
	f.Add("https://git.example.com", "HTTP://evil.example.com/token")
	f.Add("http://git.example.com", "/oauth/token")
	f.Add("https://git.example.com", "javascript:alert(1)")
	f.Add("https://git.example.com", "https:/token")
	f.Add("https://git.example.com", "")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, base, ref string) {
		resolved, err := resolveReference(base, ref)
		if err != nil {
			return
		}

		parsed, parseErr := url.Parse(resolved)
		if parseErr != nil {
			t.Fatalf("resolved %q from %q and %q does not parse: %v", resolved, base, ref, parseErr)
		}

		if parsed.Scheme != schemeHTTPS {
			t.Fatalf("resolved %q from %q and %q is not https", resolved, base, ref)
		}

		cleared, clearErr := resolveConfiguredURL(base, "")
		if clearErr != nil || cleared != "" {
			t.Fatalf("an empty configured value did not clear: %q, %v", cleared, clearErr)
		}
	})
}
