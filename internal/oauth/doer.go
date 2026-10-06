// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

// Doer sends an HTTP request.
//
// [http.Client] implements Doer. A non-nil Doer is injected through
// [oauth2.HTTPClient] so token requests use the caller's transport.
type Doer interface {
	// Do sends req and returns the response.
	//
	// Parameters:
	//   - req: request to send.
	//
	// Returns:
	//   - *[http.Response]: response. The caller closes the body.
	//   - error: non-nil when the request cannot be sent.
	Do(req *http.Request) (*http.Response, error)
}

// doerTransport adapts a Doer to an [http.RoundTripper].
type doerTransport struct {
	doer Doer
}

// requestTimeout bounds a single OAuth exchange.
//
// [http.Client] has no default timeout, so an endpoint that accepts a
// connection and then never answers would block the whole git operation
// indefinitely. The user has no way to interrupt it, because the process is
// git's child and git is waiting on the credential.
//
// A device grant already polls for as long as the grant lives, so this bounds
// one exchange rather than the grant.
const requestTimeout = 2 * time.Minute

// Compile-time check that the standard client is a Doer.
var _ Doer = (*http.Client)(nil)

// RoundTrip sends req through the wrapped Doer.
//
// Parameters:
//   - req: request to send.
//
// Returns:
//   - *[http.Response]: response from the Doer.
//   - error: non-nil when the Doer fails.
func (t doerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	return resp, nil
}

// withTimeout bounds a client that has no timeout of its own.
//
// A caller that chose a timeout keeps it. The copy matters because setting the
// field on the caller's client would change behavior for anything else sharing
// it.
//
// Parameters:
//   - client: client that may lack a timeout.
//
// Returns:
//   - *[http.Client]: client with a bounded timeout.
func withTimeout(client *http.Client) *http.Client {
	if client.Timeout != 0 {
		return client
	}

	bounded := *client

	bounded.Timeout = requestTimeout

	return &bounded
}

// withDoer returns ctx bound to doer as the oauth2 HTTP client.
//
// A nil doer leaves ctx unchanged. A [http.Client] is injected directly. Any
// other Doer is wrapped in a client whose transport calls Do. Either way the
// client that reaches oauth2 has a bounded timeout.
//
// Parameters:
//   - ctx: parent context.
//   - doer: HTTP client. Nil leaves ctx unchanged.
//
// Returns:
//   - [context.Context]: ctx carrying the oauth2 HTTP client.
func withDoer(ctx context.Context, doer Doer) context.Context {
	if doer == nil {
		return ctx
	}

	client, ok := doer.(*http.Client)
	if ok {
		return context.WithValue(ctx, oauth2.HTTPClient, withTimeout(client))
	}

	return context.WithValue(
		ctx,
		oauth2.HTTPClient,
		&http.Client{
			Transport: doerTransport{doer: doer},
			Timeout:   requestTimeout,
		},
	)
}
