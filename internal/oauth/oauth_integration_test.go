// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/nicholas-fedor/git-credential-oauth/internal/oauth"
	mockOauth "github.com/nicholas-fedor/git-credential-oauth/internal/oauth/mocks"
)

const (
	deviceUserCode     = "ABCD-EFGH"
	deviceURI          = "https://example.com/device"
	deviceComplete     = "https://example.com/device?user_code=ABCD-EFGH"
	resolvedRedirect   = "http://127.0.0.1:34567/callback"
	configuredRedirect = "http://127.0.0.1:0/callback"
)

func TestAcquireCodeGrant(t *testing.T) {
	t.Parallel()

	script, server := newScriptServer(t, false)
	browser, targetURL := openBrowser(t, nil)
	callback := queryCallback(t, resolvedRedirect, url.Values{
		"code":  {"auth-code"},
		"state": {"state-1"},
	}, nil)
	factory, redirectURL := startFactory(t, callback)
	acq := &oauth.ConfigAcquirer{
		Browser:  browser,
		Callback: factory,
		Doer:     server.Client(),
		Nonce: func() (string, error) {
			return "state-1", nil
		},
	}

	token, err := acq.Acquire(t.Context(), oauthConfig(server), oauth.Input{
		AuthURLSuffix: "&login=octocat",
		PKCE:          true,
	})
	require.NoError(t, err)
	assert.Equal(t, "access-1", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.Equal(t, "refresh-1", token.RefreshToken)
	assert.WithinDuration(t, time.Now().Add(time.Hour), token.Expiry, time.Minute)
	assert.Equal(t, configuredRedirect, *redirectURL)

	authQuery := mustQuery(t, *targetURL)
	assert.Equal(t, "state-1", authQuery.Get("state"))
	assert.Equal(t, "S256", authQuery.Get("code_challenge_method"))
	assert.Equal(t, resolvedRedirect, authQuery.Get("redirect_uri"))
	assert.Equal(t, "octocat", authQuery.Get("login"))
	assert.NotEqual(t, configuredRedirect, authQuery.Get("redirect_uri"))

	posted := script.grant("authorization_code")
	require.NotNil(t, posted)
	assert.Equal(t, "auth-code", posted.Get("code"))
	assert.Equal(t, resolvedRedirect, posted.Get("redirect_uri"))
	assert.Equal(t, oauth2.S256ChallengeFromVerifier(posted.Get("code_verifier")),
		authQuery.Get("code_challenge"))
	assert.NotEmpty(t, posted.Get("code_verifier"))
}

func TestAcquireWaitError(t *testing.T) {
	t.Parallel()

	waitErr := errors.New("duplicate request")
	callback := queryCallback(t, resolvedRedirect, nil, waitErr)
	browser, _ := openBrowser(t, nil)
	acq := codeAcquirer(t, browser, callback, nil)

	_, err := acq.Acquire(t.Context(), baseConfig(), oauth.Input{})
	require.ErrorIs(t, err, waitErr)
}

func TestAcquireStateMismatch(t *testing.T) {
	t.Parallel()

	callback := queryCallback(t, resolvedRedirect, url.Values{
		"state": {"other"},
		"code":  {"auth-code"},
	}, nil)
	browser, _ := openBrowser(t, nil)
	acq := codeAcquirer(t, browser, callback, nil)

	_, err := acq.Acquire(t.Context(), baseConfig(), oauth.Input{})
	require.ErrorIs(t, err, oauth.ErrStateMismatch)
}

func TestAcquireWaitCanceled(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	callback := blockingCallback(t, resolvedRedirect, started)
	browser, _ := openBrowser(t, nil)
	acq := codeAcquirer(t, browser, callback, nil)

	ctx, cancel := context.WithCancel(t.Context())
	errCh := make(chan error, 1)

	go func() {
		_, err := acq.Acquire(ctx, baseConfig(), oauth.Input{})
		errCh <- err
	}()

	<-started
	cancel()

	require.ErrorIs(t, <-errCh, context.Canceled)
}

func TestAcquireDeviceGrant(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		script, server := newScriptServer(t, false)
		prompter, prompted := devicePrompter(t, nil)
		acq := &oauth.ConfigAcquirer{
			Prompter: prompter,
			Doer:     server.Client(),
		}

		token, err := acq.Acquire(t.Context(), oauthConfig(server), oauth.Input{
			Device:     true,
			DeviceFlow: true,
		})
		require.NoError(t, err)
		assert.Equal(t, "access-1", token.AccessToken)
		assert.Equal(t, "refresh-1", token.RefreshToken)
		assert.Equal(t, deviceUserCode, prompted.userCode)
		assert.Equal(t, deviceURI, prompted.verificationURI)
		assert.Equal(t, deviceComplete, prompted.verificationURIComplete)
		require.NotNil(t, script.grant("urn:ietf:params:oauth:grant-type:device_code"))
	})
}

func TestAcquirePromptDeviceError(t *testing.T) {
	t.Parallel()

	promptErr := errors.New("qr encode failed")
	script, server := newScriptServer(t, false)
	prompter, _ := devicePrompter(t, promptErr)
	acq := &oauth.ConfigAcquirer{
		Prompter: prompter,
		Doer:     server.Client(),
	}

	_, err := acq.Acquire(t.Context(), oauthConfig(server), oauth.Input{
		Device:     true,
		DeviceFlow: true,
	})
	require.ErrorIs(t, err, promptErr)
	assert.Nil(t, script.grant("urn:ietf:params:oauth:grant-type:device_code"))
}

func TestAcquireDeviceUnsupported(t *testing.T) {
	t.Parallel()

	prompter := mockOauth.NewMockPrompter(t)
	doer := mockOauth.NewMockDoer(t)
	cfg := baseConfig()
	cfg.Endpoint.DeviceAuthURL = "https://example.com/device"
	cfg.Endpoint.TokenURL = "https://example.com/token"
	acq := &oauth.ConfigAcquirer{
		Prompter: prompter,
		Doer:     doer,
	}

	_, err := acq.Acquire(t.Context(), cfg, oauth.Input{
		Device:     true,
		DeviceFlow: false,
	})
	require.ErrorIs(t, err, oauth.ErrDeviceUnsupported)
	prompter.AssertNotCalled(
		t, "PromptDevice",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	)
	doer.AssertNotCalled(t, "Do", mock.Anything)
}

func TestAcquireRefreshSuccessSkipsBrowser(t *testing.T) {
	t.Parallel()

	_, server := newScriptServer(t, false)
	browser := mockOauth.NewMockBrowser(t)
	acq := &oauth.ConfigAcquirer{
		Browser: browser,
		Doer:    server.Client(),
	}

	token, err := acq.Acquire(t.Context(), oauthConfig(server), oauth.Input{
		RefreshToken: "stale-refresh",
	})
	require.NoError(t, err)
	assert.Equal(t, "access-1", token.AccessToken)
	assert.Equal(t, "refresh-1", token.RefreshToken)
	browser.AssertNotCalled(t, "Open", mock.Anything, mock.Anything)
}

func TestAcquireRefreshFailureFallsThrough(t *testing.T) {
	t.Parallel()

	script, server := newScriptServer(t, true)
	logger, warns := warnLogger(t)
	browser, _ := openBrowser(t, nil)
	callback := queryCallback(t, resolvedRedirect, url.Values{
		"code":  {"auth-code"},
		"state": {"state-1"},
	}, nil)
	factory, _ := startFactory(t, callback)
	acq := &oauth.ConfigAcquirer{
		Browser:  browser,
		Callback: factory,
		Doer:     server.Client(),
		Log:      logger,
		Nonce: func() (string, error) {
			return "state-1", nil
		},
	}

	token, err := acq.Acquire(t.Context(), oauthConfig(server), oauth.Input{
		RefreshToken: "stale-refresh",
	})
	require.NoError(t, err)
	assert.Equal(t, "access-1", token.AccessToken)
	assert.Equal(t, []string{"refresh failed"}, *warns)
	assert.NotNil(t, script.grant("refresh_token"))
	assert.NotNil(t, script.grant("authorization_code"))
}

func TestAcquireNilLoggerRefreshFailure(t *testing.T) {
	t.Parallel()

	_, server := newScriptServer(t, true)
	prompter := mockOauth.NewMockPrompter(t)
	acq := &oauth.ConfigAcquirer{
		Doer:     server.Client(),
		Prompter: prompter,
	}

	_, err := acq.Acquire(t.Context(), oauthConfig(server), oauth.Input{
		RefreshToken: "stale-refresh",
		Device:       true,
		DeviceFlow:   false,
	})
	require.ErrorIs(t, err, oauth.ErrDeviceUnsupported)
	prompter.AssertNotCalled(
		t, "PromptDevice",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	)
}

func TestAcquireNilBrowser(t *testing.T) {
	t.Parallel()

	acq := &oauth.ConfigAcquirer{}

	_, err := acq.Acquire(t.Context(), baseConfig(), oauth.Input{})
	require.ErrorContains(t, err, "browser is nil")
}

func TestAcquireNilCallback(t *testing.T) {
	t.Parallel()

	browser := mockOauth.NewMockBrowser(t)
	acq := &oauth.ConfigAcquirer{Browser: browser}

	_, err := acq.Acquire(t.Context(), baseConfig(), oauth.Input{})
	require.ErrorContains(t, err, "callback factory is nil")
	browser.AssertNotCalled(t, "Open", mock.Anything, mock.Anything)
}

func TestAcquireBrowserOpenError(t *testing.T) {
	t.Parallel()

	openErr := errors.New("open failed")
	callback := redirectCallback(t, resolvedRedirect)
	browser, targetURL := openBrowser(t, openErr)
	acq := codeAcquirer(t, browser, callback, nil)

	_, err := acq.Acquire(t.Context(), baseConfig(), oauth.Input{})
	require.ErrorIs(t, err, openErr)
	assert.NotEmpty(t, *targetURL)
}

func TestAcquireAuthorizationError(t *testing.T) {
	t.Parallel()

	callback := queryCallback(t, resolvedRedirect, url.Values{
		"error": {"access_denied"},
		"state": {"state-1"},
	}, nil)
	browser, _ := openBrowser(t, nil)
	acq := codeAcquirer(t, browser, callback, nil)

	_, err := acq.Acquire(t.Context(), baseConfig(), oauth.Input{})
	require.ErrorContains(t, err, "access_denied")
}

func codeAcquirer(
	tb testing.TB,
	browser oauth.Browser,
	callback oauth.Callback,
	doer oauth.Doer,
) *oauth.ConfigAcquirer {
	tb.Helper()

	if doer == nil {
		doer = mockOauth.NewMockDoer(tb)
	}

	factory, _ := startFactory(tb, callback)

	return &oauth.ConfigAcquirer{
		Browser:  browser,
		Callback: factory,
		Doer:     doer,
		Nonce: func() (string, error) {
			return "state-1", nil
		},
	}
}

func redirectCallback(tb testing.TB, redirect string) *mockOauth.MockCallback {
	tb.Helper()

	callback := mockOauth.NewMockCallback(tb)
	callback.EXPECT().RedirectURL().Return(redirect)
	callback.EXPECT().Close().Return(nil)

	return callback
}

func queryCallback(
	tb testing.TB,
	redirect string,
	query url.Values,
	waitErr error,
) *mockOauth.MockCallback {
	tb.Helper()

	callback := mockOauth.NewMockCallback(tb)
	callback.EXPECT().RedirectURL().Return(redirect)
	callback.EXPECT().Wait(mock.Anything).Return(query, waitErr)
	callback.EXPECT().Close().Return(nil)

	return callback
}

func blockingCallback(
	tb testing.TB,
	redirect string,
	started chan struct{},
) *mockOauth.MockCallback {
	tb.Helper()

	callback := mockOauth.NewMockCallback(tb)
	callback.EXPECT().RedirectURL().Return(redirect)
	callback.EXPECT().Wait(mock.Anything).RunAndReturn(func(ctx context.Context) (url.Values, error) {
		close(started)
		<-ctx.Done()

		return nil, ctx.Err()
	})
	callback.EXPECT().Close().Return(nil)

	return callback
}

func startFactory(
	tb testing.TB,
	callback oauth.Callback,
) (*mockOauth.MockCallbackFactory, *string) {
	tb.Helper()

	factory := mockOauth.NewMockCallbackFactory(tb)
	redirectURL := ""
	factory.EXPECT().Start(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, redirect string) (oauth.Callback, error) {
			redirectURL = redirect

			return callback, nil
		},
	).Once()

	return factory, &redirectURL
}

func openBrowser(tb testing.TB, openErr error) (*mockOauth.MockBrowser, *string) {
	tb.Helper()

	browser := mockOauth.NewMockBrowser(tb)
	targetURL := ""
	browser.EXPECT().Open(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, target string) error {
			targetURL = target

			return openErr
		},
	).Once()

	return browser, &targetURL
}

type devicePrompt struct {
	userCode                string
	verificationURI         string
	verificationURIComplete string
}

func devicePrompter(tb testing.TB, promptErr error) (*mockOauth.MockPrompter, *devicePrompt) {
	tb.Helper()

	prompter := mockOauth.NewMockPrompter(tb)
	prompted := &devicePrompt{}
	prompter.EXPECT().
		PromptDevice(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			userCode, verificationURI, verificationURIComplete string,
		) error {
			prompted.userCode = userCode
			prompted.verificationURI = verificationURI
			prompted.verificationURIComplete = verificationURIComplete

			return promptErr
		}).
		Once()

	return prompter, prompted
}

func warnLogger(tb testing.TB) (*mockOauth.MockLogger, *[]string) {
	tb.Helper()

	logger := mockOauth.NewMockLogger(tb)
	warns := []string{}
	logger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything).Return()
	logger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything).Run(
		func(_ context.Context, msg string, _ ...any) {
			warns = append(warns, msg)
		},
	).Return()

	return logger, &warns
}

func baseConfig() oauth2.Config {
	return oauth2.Config{
		ClientID:    "client-id",
		RedirectURL: configuredRedirect,
		Scopes:      []string{"repo"},
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://example.com/login/oauth/authorize",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}

func oauthConfig(server *httptest.Server) oauth2.Config {
	cfg := baseConfig()
	cfg.ClientSecret = "client-secret"
	cfg.Endpoint.TokenURL = server.URL + "/token"
	cfg.Endpoint.DeviceAuthURL = server.URL + "/device"

	return cfg
}

func mustQuery(t *testing.T, rawURL string) url.Values {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)

	return parsed.Query()
}

type recordedRequest struct {
	values url.Values
	path   string
}

type script struct {
	reqs        []recordedRequest
	mu          sync.Mutex
	failRefresh bool
}

func newScriptServer(t *testing.T, failRefresh bool) (*script, *httptest.Server) {
	t.Helper()

	handler := &script{failRefresh: failRefresh}

	return handler, httptest.NewTestServer(t, handler)
}

func (s *script) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	s.mu.Lock()
	s.reqs = append(s.reqs, recordedRequest{path: r.URL.Path, values: values})
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/device":
		_, _ = io.WriteString(w, `{
			"device_code":"device-code",
			"user_code":"`+deviceUserCode+`",
			"verification_uri":"`+deviceURI+`",
			"verification_uri_complete":"`+deviceComplete+`",
			"expires_in":900,
			"interval":1
		}`)
	case "/token":
		if s.failRefresh && values.Get("grant_type") == "refresh_token" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)

			return
		}

		_, _ = io.WriteString(w, `{
			"access_token":"access-1",
			"token_type":"Bearer",
			"refresh_token":"refresh-1",
			"expires_in":3600
		}`)
	default:
		http.NotFound(w, r)
	}
}

func (s *script) grant(grantType string) url.Values {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, req := range s.reqs {
		if req.values.Get("grant_type") == grantType {
			return req.values
		}
	}

	return nil
}
