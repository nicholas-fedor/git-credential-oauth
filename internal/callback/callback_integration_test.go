// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package callback_test exercises the loopback server through its exported API.
package callback_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/callback"
)

const integrationTimeout = 5 * time.Second

// TestFactoryDeliversTheRedirectOnce runs a whole redirect against a real
// loopback listener, as the browser grant does.
func TestFactoryDeliversTheRedirectOnce(t *testing.T) {
	t.Parallel()

	server, err := (&callback.Factory{Version: "v9.9.9"}).Start(t.Context(), "")
	require.NoError(t, err)

	t.Cleanup(func() { _ = server.Close() })

	target := server.RedirectURL() + "/?" + url.Values{
		"code":  {"auth-code"},
		"state": {"state-1"},
	}.Encode()

	body := fetch(t, target)
	assert.Contains(t, body, "return to Git")
	assert.Contains(t, body, "v9.9.9")

	ctx, cancel := context.WithTimeout(t.Context(), integrationTimeout)
	defer cancel()

	query, err := server.Wait(ctx)
	require.NoError(t, err)
	assert.Equal(t, "auth-code", query.Get("code"))
	assert.Equal(t, "state-1", query.Get("state"))
}

// TestWaitReturnsAfterClose checks a closed server releases its waiter.
func TestWaitReturnsAfterClose(t *testing.T) {
	t.Parallel()

	server, err := (&callback.Factory{}).Start(t.Context(), "")
	require.NoError(t, err)
	require.NoError(t, server.Close())

	ctx, cancel := context.WithTimeout(t.Context(), integrationTimeout)
	defer cancel()

	_, err = server.Wait(ctx)
	require.Error(t, err)
	require.NoError(t, server.Close(), "a second Close is harmless")
}

// TestStartRejectsARedirectWithoutAHost covers a misconfigured redirect.
func TestStartRejectsARedirectWithoutAHost(t *testing.T) {
	t.Parallel()

	server, err := (&callback.Factory{}).Start(t.Context(), "/callback")
	require.Error(t, err)
	assert.Nil(t, server)
}

// fetch performs a GET and returns the body.
func fetch(t *testing.T, target string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), integrationTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return string(body)
}
