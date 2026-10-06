// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// stubCallback is a callback that returns a fixed query.
type stubCallback struct {
	waitErr  error
	closeErr error
	query    url.Values
	redirect string
	closed   int
}

func (c *stubCallback) RedirectURL() string { return c.redirect }

// Wait returns the configured query or error.
func (c *stubCallback) Wait(context.Context) (url.Values, error) {
	if c.waitErr != nil {
		return nil, c.waitErr
	}

	return c.query, nil
}

// Close counts the call and returns the configured error.
func (c *stubCallback) Close() error {
	c.closed++

	return c.closeErr
}

// stubFactory starts a fixed callback.
type stubFactory struct {
	callback Callback
	err      error
}

// Start returns the configured callback or error.
//

func (f stubFactory) Start(context.Context, string) (Callback, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.callback, nil
}

// stubBrowser records the URL it was asked to open.
type stubBrowser struct {
	err    error
	opened string
}

// Open records targetURL.
func (b *stubBrowser) Open(_ context.Context, targetURL string) error {
	b.opened = targetURL

	return b.err
}

var (
	errStub  = errors.New("stub failure")
	errClose = errors.New("close failure")
)

// TestAuthCodeRequestURLAlwaysSendsS256 checks PKCE cannot be skipped.
func TestAuthCodeRequestURLAlwaysSendsS256(t *testing.T) {
	t.Parallel()

	cfg := oauth2.Config{
		ClientID:    "id",
		Endpoint:    oauth2.Endpoint{AuthURL: "https://git.example.com/authorize"},
		RedirectURL: "http://127.0.0.1:1",
		Scopes:      []string{"a", "b"},
	}

	raw := AuthCodeRequestURL(cfg, "state-1", "verifier-1234567890-1234567890-1234567890", "&login=octocat")

	parsed, err := url.Parse(raw)
	require.NoError(t, err)

	query := parsed.Query()
	assert.Equal(t, "S256", query.Get("code_challenge_method"))
	assert.Equal(t, oauth2.S256ChallengeFromVerifier("verifier-1234567890-1234567890-1234567890"),
		query.Get("code_challenge"))
	assert.Equal(t, "state-1", query.Get("state"))
	assert.Equal(t, "id", query.Get("client_id"))
	assert.Equal(t, "a b", query.Get("scope"))
	assert.Equal(t, "octocat", query.Get("login"), "the suffix is appended as query text")
}

// TestWaitCodeQuery covers every way the redirect can be refused.
func TestWaitCodeQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr  error
		callback *stubCallback
		name     string
		wantCode string
	}{
		{
			name:     "matching state and a code",
			callback: &stubCallback{query: url.Values{"state": {"s"}, "code": {"c"}}},
			wantCode: "c",
		},
		{
			name:     "a different state",
			callback: &stubCallback{query: url.Values{"state": {"other"}, "code": {"c"}}},
			wantErr:  ErrStateMismatch,
		},
		{
			name:     "no state at all",
			callback: &stubCallback{query: url.Values{"code": {"c"}}},
			wantErr:  ErrStateMismatch,
		},
		{
			name:     "the forge reports an error",
			callback: &stubCallback{query: url.Values{"state": {"s"}, "error": {"access_denied"}}},
			wantErr:  errAuthorization,
		},
		{
			name:     "no code",
			callback: &stubCallback{query: url.Values{"state": {"s"}}},
			wantErr:  errMissingCode,
		},
		{
			name:     "the wait fails",
			callback: &stubCallback{waitErr: errStub},
			wantErr:  errStub,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code, err := waitCodeQuery(t.Context(), tt.callback, "s")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, code)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantCode, code)
		})
	}
}

// TestWaitCodeQueryChecksStateBeforeTheError keeps a forged error out.
//
// A request without the right state is not this sign-in's redirect, so its
// error parameter must not be reported as the forge's answer.
func TestWaitCodeQueryChecksStateBeforeTheError(t *testing.T) {
	t.Parallel()

	_, err := waitCodeQuery(t.Context(), &stubCallback{
		query: url.Values{"state": {"forged"}, "error": {"access_denied"}},
	}, "s")
	require.ErrorIs(t, err, ErrStateMismatch)
}

// TestStartCodeCallback covers the start failures.
func TestStartCodeCallback(t *testing.T) {
	t.Parallel()

	_, err := startCodeCallback(t.Context(), stubFactory{err: errStub}, "")
	require.ErrorIs(t, err, errStub)

	_, err = startCodeCallback(t.Context(), stubFactory{}, "")
	require.ErrorIs(t, err, errNilCallback)

	callback := &stubCallback{redirect: "http://127.0.0.1:1"}
	got, err := startCodeCallback(t.Context(), stubFactory{callback: callback}, "")
	require.NoError(t, err)
	assert.Same(t, callback, got)
}

// TestOpenCodeBrowserUsesTheListenerAddress checks the redirect sent is the
// bound one.
//
// The configured redirect may name port 0. The forge has to be told the port
// that is actually listening, or it redirects to nothing.
func TestOpenCodeBrowserUsesTheListenerAddress(t *testing.T) {
	t.Parallel()

	browser := &stubBrowser{}
	cfg := oauth2.Config{
		Endpoint:    oauth2.Endpoint{AuthURL: "https://git.example.com/authorize"},
		RedirectURL: "http://127.0.0.1:0/cb",
	}
	callback := &stubCallback{redirect: "http://127.0.0.1:4242/cb"}

	resolved, err := openCodeBrowser(t.Context(), browser, cfg, callback, "s", oauth2.GenerateVerifier(), "")
	require.NoError(t, err)

	assert.Equal(t, "http://127.0.0.1:4242/cb", resolved.RedirectURL)
	assert.Equal(t, "http://127.0.0.1:0/cb", cfg.RedirectURL, "the caller's config is not changed")

	parsed, err := url.Parse(browser.opened)
	require.NoError(t, err)
	assert.Equal(t, "http://127.0.0.1:4242/cb", parsed.Query().Get("redirect_uri"))

	_, err = openCodeBrowser(t.Context(), &stubBrowser{err: errStub}, cfg, callback, "s", "v", "")
	require.ErrorIs(t, err, errStub)
}

// TestAcquireCodeClosesTheCallbackOnEveryPath checks the listener never leaks.
func TestAcquireCodeClosesTheCallbackOnEveryPath(t *testing.T) {
	t.Parallel()

	for name, acq := range map[string]func(*stubCallback) *ConfigAcquirer{
		"browser fails": func(callback *stubCallback) *ConfigAcquirer {
			return &ConfigAcquirer{Browser: &stubBrowser{err: errStub}, Callback: stubFactory{callback: callback}}
		},
		"wait fails": func(callback *stubCallback) *ConfigAcquirer {
			callback.waitErr = errStub

			return &ConfigAcquirer{Browser: &stubBrowser{}, Callback: stubFactory{callback: callback}}
		},
	} {
		callback := &stubCallback{redirect: "http://127.0.0.1:1", closeErr: errClose}

		_, err := acq(callback).acquireCode(t.Context(), oauth2.Config{}, Input{})
		require.ErrorIs(t, err, errStub, name)
		require.ErrorIs(t, err, errClose, "%s: the close error is reported too", name)
		assert.Equal(t, 1, callback.closed, name)
	}
}

// TestAcquireCodeStopsOnAStateFailure covers a broken nonce source.
func TestAcquireCodeStopsOnAStateFailure(t *testing.T) {
	t.Parallel()

	acq := &ConfigAcquirer{
		Browser:  &stubBrowser{},
		Callback: stubFactory{err: errStub},
		Nonce:    func() (string, error) { return "", errClose },
	}

	_, err := acq.acquireCode(t.Context(), oauth2.Config{}, Input{})
	require.ErrorIs(t, err, errClose)
}

// TestExchangeCodeSendsTheVerifier checks the PKCE proof reaches the forge.
func TestExchangeCodeSendsTheVerifier(t *testing.T) {
	t.Parallel()

	var form url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		form = r.PostForm

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"bearer","refresh_token":"ref"}`))
	}))
	t.Cleanup(server.Close)

	cfg := oauth2.Config{
		ClientID: "id",
		Endpoint: oauth2.Endpoint{TokenURL: server.URL, AuthStyle: oauth2.AuthStyleInParams},
	}

	token, err := exchangeCode(t.Context(), cfg, server.Client(), "code-1", "verifier-1")
	require.NoError(t, err)

	assert.Equal(t, "tok", token.AccessToken)
	assert.Equal(t, "ref", token.RefreshToken)
	assert.Equal(t, "code-1", form.Get("code"))
	assert.Equal(t, "verifier-1", form.Get("code_verifier"))
	assert.Empty(t, form.Get("client_secret"), "no secret is sent when none is configured")
}

// TestExchangeCodeReportsARejection covers a forge that refuses the code.
func TestExchangeCodeReportsARejection(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	t.Cleanup(server.Close)

	cfg := oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: server.URL, AuthStyle: oauth2.AuthStyleInParams}}

	_, err := exchangeCode(t.Context(), cfg, server.Client(), "code-1", "verifier-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exchange code")
	assert.Contains(t, err.Error(), "invalid_grant")
}
