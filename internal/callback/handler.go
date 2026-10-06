// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package callback

import (
	"net/http"
	"net/url"
)

const (
	// headerContentType is the response header that carries the page type.
	headerContentType = "Content-Type"

	// contentTypeHTML is the media type of the success page.
	contentTypeHTML = "text/html"
)

// newCallbackHandler returns the loopback request handler.
//
// Every request writes the success page. Only the first query is kept.
//
// Parameters:
//   - queries: buffered channel that receives the first callback query.
//   - version: application version written into the success page.
//
// Returns:
//   - [http.HandlerFunc]: handler for the loopback listener.
func newCallbackHandler(
	queries chan<- url.Values,
	version string,
) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		sendQuery(queries, req.URL.Query())
		writePage(resp, version)
	}
}

// sendQuery records a callback query without blocking the handler.
//
// The send uses a select with a default case. A full buffer means a callback
// is already waiting, so this request is dropped instead of stalling.
//
// Parameters:
//   - queries: one-slot channel owned by the server.
//   - query: parsed query from the current request.
func sendQuery(queries chan<- url.Values, query url.Values) {
	select {
	case queries <- query:
	default:
		// The one-slot buffer already holds a callback.
	}
}

// writePage writes the HTML success page.
//
// A write error is ignored because the browser connection is already failing.
//
// Parameters:
//   - resp: response writer for the callback request.
//   - version: application version written into the page.
func writePage(resp http.ResponseWriter, version string) {
	resp.Header().Set(headerContentType, contentTypeHTML)

	_, err := resp.Write(successPage(version))
	if err != nil {
		return
	}
}
