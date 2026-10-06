// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package callback

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testVersion = "v1.2.3-test"
	getTimeout  = 5 * time.Second
)

func TestStartSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	server := startServer(t, ctx, "")

	if !regexp.MustCompile(`^http://127\.0\.0\.1:\d+$`).MatchString(server.RedirectURL()) {
		t.Fatalf("RedirectURL() = %q, want http://127.0.0.1:<port>", server.RedirectURL())
	}

	query := url.Values{}
	query.Set("code", "abc")
	query.Set("state", "xyz")

	status, body := doGet(t, ctx, callbackURL(t, server.RedirectURL(), query))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}

	if !strings.Contains(body, testVersion) {
		t.Fatalf("body missing version %q: %s", testVersion, body)
	}

	if !strings.Contains(body, "https://github.com/nicholas-fedor/git-credential-oauth") {
		t.Fatalf("body missing repository link: %s", body)
	}

	waitCtx, cancel := context.WithTimeout(ctx, getTimeout)
	defer cancel()

	got, err := server.Wait(waitCtx)
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}

	if got.Get("code") != "abc" || got.Get("state") != "xyz" {
		t.Fatalf("Wait() = %v, want code=abc state=xyz", got)
	}
}

func TestDuplicateRequestDoesNotBlock(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	server := startServer(t, ctx, "")

	first := url.Values{}
	first.Set("code", "first")

	second := url.Values{}
	second.Set("code", "second")

	status, body := doGet(t, ctx, callbackURL(t, server.RedirectURL(), first))
	if status != http.StatusOK {
		t.Fatalf("first status = %d, want %d", status, http.StatusOK)
	}

	if !strings.Contains(body, testVersion) {
		t.Fatalf("first body missing version: %s", body)
	}

	status, body = doGet(t, ctx, callbackURL(t, server.RedirectURL(), second))
	if status != http.StatusOK {
		t.Fatalf("second status = %d, want %d", status, http.StatusOK)
	}

	if !strings.Contains(body, "https://github.com/nicholas-fedor/git-credential-oauth") {
		t.Fatalf("second body missing repository link: %s", body)
	}

	waitCtx, cancel := context.WithTimeout(ctx, getTimeout)
	defer cancel()

	got, err := server.Wait(waitCtx)
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}

	if got.Get("code") != "first" {
		t.Fatalf("Wait() code = %q, want first", got.Get("code"))
	}

	third := url.Values{}
	third.Set("code", "third")

	status, _ = doGet(t, ctx, callbackURL(t, server.RedirectURL(), third))
	if status != http.StatusOK {
		t.Fatalf("third status = %d, want %d", status, http.StatusOK)
	}
}

func TestWaitUnblocksOnCancel(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	server := startServer(t, ctx, "")

	waitCtx, cancel := context.WithCancel(ctx)
	errCh := make(chan error, 1)

	go func() {
		_, waitErr := server.Wait(waitCtx)
		errCh <- waitErr
	}()

	cancel()

	select {
	case waitErr := <-errCh:
		if !errors.Is(waitErr, context.Canceled) {
			t.Fatalf("Wait() error = %v, want context.Canceled", waitErr)
		}
	case <-time.After(getTimeout):
		t.Fatal("Wait did not return after cancel")
	}
}

func TestRedirectURLPreservesConfiguredHost(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	server := startServer(t, ctx, "http://localhost/callback")

	got := server.RedirectURL()
	if !regexp.MustCompile(`^http://localhost:\d+/callback$`).MatchString(got) {
		t.Fatalf("RedirectURL() = %q, want http://localhost:<port>/callback", got)
	}

	if strings.Contains(got, "127.0.0.1") {
		t.Fatalf("RedirectURL() = %q, want configured hostname", got)
	}

	status, body := doGet(t, ctx, got)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}

	if !strings.Contains(body, testVersion) {
		t.Fatalf("body missing version: %s", body)
	}
}

func TestCloseIdempotent(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	server := startServer(t, ctx, "")

	errCh := make(chan error, 1)

	go func() {
		_, waitErr := server.Wait(ctx)
		errCh <- waitErr
	}()

	first := server.Close()
	second := server.Close()

	if first != nil {
		t.Fatalf("first Close() error = %v", first)
	}

	if second != nil {
		t.Fatalf("second Close() error = %v", second)
	}

	if !errors.Is(first, second) {
		t.Fatalf("Close() errors differ: %v and %v", first, second)
	}

	select {
	case waitErr := <-errCh:
		if !errors.Is(waitErr, errServerClosed) {
			t.Fatalf("Wait() error = %v, want errServerClosed", waitErr)
		}
	case <-time.After(getTimeout):
		t.Fatal("Wait did not return after Close")
	}
}

// TestCloseStopsAcceptingConnections checks Close leaves nothing listening.
//
// The accept loop is the only goroutine this package starts, and Close joins it
// before returning. Once the listener is gone the loop cannot be running, which
// is the property a caller depends on: a credential request must not leave a
// goroutine behind on every authorization.
func TestCloseStopsAcceptingConnections(t *testing.T) {
	t.Parallel()

	server := startServer(t, t.Context(), "")
	address := server.listener.Addr().String()

	require.NoError(t, server.Close())

	dialer := &net.Dialer{Timeout: getTimeout}

	conn, dialErr := dialer.DialContext(t.Context(), "tcp", address)
	if dialErr == nil {
		_ = conn.Close()
		t.Fatalf("listener at %s still accepts after Close", address)
	}
}

func TestStartBindFailure(t *testing.T) {
	t.Parallel()

	factory := &Factory{Version: testVersion}

	server, err := factory.Start(t.Context(), "http://127.0.0.1:99999/callback")
	if err == nil {
		t.Fatal("Start() error = nil, want bind failure")
	}

	if server != nil {
		t.Fatalf("Start() server = %#v, want nil", server)
	}
}

func startServer(t *testing.T, ctx context.Context, redirectURL string) *Server {
	t.Helper()

	factory := &Factory{Version: testVersion}

	server, err := factory.Start(ctx, redirectURL)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	t.Cleanup(func() {
		closeErr := server.Close()
		if closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})

	return server
}

func callbackURL(t *testing.T, redirectURL string, query url.Values) string {
	t.Helper()

	parsed, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}

	parsed.RawQuery = query.Encode()

	return parsed.String()
}

func doGet(t *testing.T, ctx context.Context, rawURL string) (int, string) {
	t.Helper()

	client := new(http.Client)
	client.Timeout = getTimeout

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", rawURL, err)
	}

	t.Cleanup(func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			t.Errorf("close body: %v", closeErr)
		}
	})

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if got := resp.Header.Get(headerContentType); got != contentTypeHTML {
		t.Fatalf("Content-Type = %q, want %q", got, contentTypeHTML)
	}

	return resp.StatusCode, string(body)
}
