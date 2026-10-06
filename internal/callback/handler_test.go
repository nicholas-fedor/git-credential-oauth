// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package callback

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingWriter is a response writer whose body writes fail.
type failingWriter struct {
	header http.Header
}

func (w *failingWriter) Header() http.Header       { return w.header }
func (w *failingWriter) WriteHeader(int)           {}
func (w *failingWriter) Write([]byte) (int, error) { return 0, errWriteFailed }

var errWriteFailed = errors.New("write failed")

// TestHandlerForwardsTheQueryAndAnswersWithThePage covers one redirect.
func TestHandlerForwardsTheQueryAndAnswersWithThePage(t *testing.T) {
	t.Parallel()

	queries := make(chan url.Values, 1)
	handler := newCallbackHandler(queries, "v1")

	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/cb?code=c&state=s", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "text/html", recorder.Header().Get("Content-Type"))
	assert.Equal(t, string(successPage("v1")), recorder.Body.String())

	got := <-queries
	assert.Equal(t, "c", got.Get("code"))
	assert.Equal(t, "s", got.Get("state"))
}

// TestHandlerKeepsOnlyTheFirstRedirect checks a second request cannot replace it.
//
// The channel holds one value. A later request, such as a browser retry or
// another local program, must not block the server or overwrite the first
// redirect the grant is waiting for.
func TestHandlerKeepsOnlyTheFirstRedirect(t *testing.T) {
	t.Parallel()

	queries := make(chan url.Values, 1)
	handler := newCallbackHandler(queries, "v1")

	for _, state := range []string{"first", "second", "third"} {
		recorder := httptest.NewRecorder()
		handler(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/?state="+state, nil))
		assert.Equal(t, http.StatusOK, recorder.Code)
	}

	assert.Equal(t, "first", (<-queries).Get("state"))
	assert.Empty(t, queries)
}

// TestSendQueryNeverBlocks checks a full channel drops the value.
func TestSendQueryNeverBlocks(t *testing.T) {
	t.Parallel()

	queries := make(chan url.Values)

	assert.NotPanics(t, func() { sendQuery(queries, url.Values{}) })
}

// TestWritePageToleratesAFailedWrite covers a browser that hung up.
func TestWritePageToleratesAFailedWrite(t *testing.T) {
	t.Parallel()

	writer := &failingWriter{header: http.Header{}}

	require.NotPanics(t, func() { writePage(writer, "v1") })
	assert.Equal(t, "text/html", writer.header.Get("Content-Type"))
}
