// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package callback

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// FuzzCallbackHandler checks the loopback handler on arbitrary query strings.
//
// Any local program can send the callback server a request, so its query is
// untrusted. Whatever it contains, the handler must answer with the fixed page,
// must not block, and must pass on exactly the query the request carried, so
// the grant's state check sees what was really sent.
//
// Parameters:
//   - f: fuzzing handle.
func FuzzCallbackHandler(f *testing.F) {
	f.Add("code=abc&state=xyz")
	f.Add("error=access_denied&state=xyz")
	f.Add("state=a&state=b&code=c")
	f.Add("code=%zz")
	f.Add(";;;&&&==")
	f.Add("code=" + "a\x00b")
	f.Add("")

	f.Fuzz(func(t *testing.T, rawQuery string) {
		queries := make(chan url.Values, 1)
		handler := newCallbackHandler(queries, "v1")

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.URL.RawQuery = rawQuery

		recorder := httptest.NewRecorder()
		handler(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status %d for query %q", recorder.Code, rawQuery)
		}

		if recorder.Body.String() != string(successPage("v1")) {
			t.Fatalf("page changed for query %q", rawQuery)
		}

		select {
		case got := <-queries:
			want := req.URL.Query()
			if got.Encode() != want.Encode() {
				t.Fatalf("forwarded %v, want %v", got, want)
			}
		default:
			t.Fatalf("query %q was not forwarded", rawQuery)
		}
	})
}
