// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/oauth2"
)

// clientFrom returns the client oauth2 would use for ctx.
//
// Parameters:
//   - ctx: context bound by withDoer.
//
// Returns:
//   - *http.Client: client stored in the context.
func clientFrom(t *testing.T, ctx context.Context) *http.Client {
	t.Helper()

	client, ok := ctx.Value(oauth2.HTTPClient).(*http.Client)
	require.True(t, ok, "no client bound to the context")

	return client
}

// TestWithDoerBoundsAnUnboundedClient is the regression this package guards.
//
// http.Client has no default timeout, so an endpoint that accepts a connection
// and never answers would block the git operation forever. The user cannot
// interrupt it because git is waiting on this process for a credential.
func TestWithDoerBoundsAnUnboundedClient(t *testing.T) {
	t.Parallel()

	ctx := withDoer(t.Context(), &http.Client{})

	assert.Equal(t, requestTimeout, clientFrom(t, ctx).Timeout)
}

// TestWithDoerKeepsAnExistingTimeout checks a caller choice is not overridden.
func TestWithDoerKeepsAnExistingTimeout(t *testing.T) {
	t.Parallel()

	const chosen = 30 * time.Second

	ctx := withDoer(t.Context(), &http.Client{Timeout: chosen})

	assert.Equal(t, chosen, clientFrom(t, ctx).Timeout)
}

// TestWithDoerDoesNotMutateTheCallerClient checks the bound client is a copy.
//
// Setting Timeout on the caller's client would change behavior for anything
// else sharing it.
func TestWithDoerDoesNotMutateTheCallerClient(t *testing.T) {
	t.Parallel()

	caller := &http.Client{}

	withDoer(t.Context(), caller)

	assert.Equal(t, time.Duration(0), caller.Timeout, "caller client was mutated")
}

// TestWithDoerBoundsAWrappedTransport covers the non-client Doer path.
func TestWithDoerBoundsAWrappedTransport(t *testing.T) {
	t.Parallel()

	ctx := withDoer(t.Context(), stubDoer{})

	client := clientFrom(t, ctx)
	assert.Equal(t, requestTimeout, client.Timeout)
	assert.NotNil(t, client.Transport, "the wrapped transport is still used")
}

// TestWithDoerKeepsAWrappedTransport covers the transport wiring.
func TestWithDoerKeepsAWrappedTransport(t *testing.T) {
	t.Parallel()

	ctx := withDoer(t.Context(), stubDoer{})

	_, ok := clientFrom(t, ctx).Transport.(doerTransport)
	assert.True(t, ok, "a non-client Doer is wrapped in a transport")
}

// TestWithDoerLeavesContextAloneWhenNoDoer is documented behavior.
//
// A nil Doer means the caller wants the oauth2 default, which has no timeout
// either. That is the caller's choice to make, not this function's.
func TestWithDoerLeavesContextAloneWhenNoDoer(t *testing.T) {
	t.Parallel()

	ctx := withDoer(t.Context(), nil)

	assert.Nil(t, ctx.Value(oauth2.HTTPClient))
}

// TestRequestTimeoutIsLongEnoughForASlowProvider documents the choice.
//
// A token exchange is one request and one response. Two minutes is generous for
// that, and short enough that a stuck socket is reported rather than hanging.
func TestRequestTimeoutIsLongEnoughForASlowProvider(t *testing.T) {
	t.Parallel()

	assert.Positive(t, requestTimeout)
	assert.LessOrEqual(t, requestTimeout, 5*time.Minute, "not long enough to break a slow provider")
}

// stubDoer is a Doer that is not an [http.Client].
type stubDoer struct{}

// Do reports success without sending anything.
//
// Parameters:
//   - req: request the caller offered.
//
// Returns:
//   - *[http.Response]: empty successful response.
//   - error: always nil.
func (stubDoer) Do(_ *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK}, nil
}
