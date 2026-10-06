// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package callback

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Server is the loopback HTTP server for one OAuth redirect.
//
// Close is idempotent. It closes a done channel to unblock Wait, shuts down the
// HTTP server, and waits for the accept loop to exit. The query channel stays
// open so a handler send cannot panic if it races Close.
type Server struct {
	listener    net.Listener
	closeErr    error
	httpServer  *http.Server
	queries     chan url.Values
	done        chan struct{}
	redirectURL string
	closeOnce   sync.Once
	acceptDone  sync.WaitGroup
}

const (
	// readHeaderTimeout limits how long the server waits for request headers.
	readHeaderTimeout = 10 * time.Second

	// readTimeout limits how long the server waits for a full request.
	readTimeout = 10 * time.Second

	// writeTimeout limits how long the server waits for a response write.
	writeTimeout = 10 * time.Second

	// idleTimeout limits how long a keep-alive connection may sit idle.
	idleTimeout = 60 * time.Second
)

// errServerClosed is returned by Wait when Close runs before a callback.
var errServerClosed = errors.New("callback server closed")

// newServer starts the accept loop and returns the running server.
//
// The query channel has one slot. The goroutine started here is the only
// accept loop for this callback, and Close waits for it, so its lifetime is
// bounded by the server rather than by the process.
//
// Parameters:
//   - listener: bound loopback listener.
//   - redirectURL: uRL reported to the OAuth provider.
//   - version: application version written into the success page.
//
// Returns:
//   - *Server: running callback server.
func newServer(listener net.Listener, redirectURL, version string) *Server {
	httpServer := new(http.Server)

	httpServer.ReadHeaderTimeout = readHeaderTimeout
	httpServer.ReadTimeout = readTimeout
	httpServer.WriteTimeout = writeTimeout
	httpServer.IdleTimeout = idleTimeout

	server := new(Server)

	server.listener = listener
	server.redirectURL = redirectURL
	server.httpServer = httpServer
	server.queries = make(chan url.Values, 1)
	server.done = make(chan struct{})
	httpServer.Handler = newCallbackHandler(server.queries, version)

	server.acceptDone.Go(server.serve)

	return server
}

// Close stops the listener and unblocks Wait.
//
// The first call closes a done channel and shuts down the HTTP server. Later
// calls return the same error. The query channel is left open.
//
// Returns:
//   - error: the error from the first shutdown, or nil.
func (srv *Server) Close() error {
	srv.closeOnce.Do(srv.shutdown)

	return srv.closeErr
}

// RedirectURL returns the loopback URL registered with the provider.
//
// When no redirect URL was configured, the result uses scheme http, host
// 127.0.0.1, and the bound port. When a redirect URL was configured, the
// result keeps that hostname and path and uses the bound port.
//
// Returns:
//   - string: the resolved redirect URL.
func (srv *Server) RedirectURL() string {
	return srv.redirectURL
}

// Wait blocks until the first callback query arrives, ctx is done, or Close
// runs.
//
// The receive is a select, not a bare receive. On cancel, Wait returns
// a wrapped ctx.Err. Close is observed on a done channel rather than a closed
// query channel, so this call does not panic if it races the handler.
//
// Parameters:
//   - ctx: canceled to stop waiting.
//
// Returns:
//   - [url.Values]: the first callback query.
//   - error: wrapped ctx.Err when ctx is canceled, or a closed-server error
//     when Close runs first.
func (srv *Server) Wait(ctx context.Context) (url.Values, error) {
	select {
	case query := <-srv.queries:
		return query, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for callback: %w", ctx.Err())
	case <-srv.done:
		return nil, errServerClosed
	}
}

// serve accepts callback connections until Close.
//
// [http.ErrServerClosed] is the expected result of Close. Other accept errors
// end the goroutine without exiting the process.
func (srv *Server) serve() {
	err := srv.httpServer.Serve(srv.listener)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return
	}
}

// shutdown closes the done channel and the HTTP server, then joins the accept
// loop.
//
// Closing done unblocks Wait. Closing the HTTP server makes Serve return, and
// waiting for it is what keeps a goroutine from outliving the server it belongs
// to. The query channel is not closed.
func (srv *Server) shutdown() {
	close(srv.done)

	err := srv.httpServer.Close()
	if err != nil {
		srv.closeErr = fmt.Errorf("close callback server: %w", err)
	}

	srv.acceptDone.Wait()
}
