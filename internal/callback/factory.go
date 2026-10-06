// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package callback

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
)

// Factory starts callback servers.
type Factory struct {
	// Version is copied into the HTML success page when Start runs.
	Version string
}

const (
	// schemeHTTP is the URL scheme for an ephemeral loopback redirect.
	schemeHTTP = "http"

	// networkTCP is the network passed to [net.ListenConfig.Listen].
	networkTCP = "tcp"

	// loopbackHost is the bind address when no redirect URL is configured.
	loopbackHost = "127.0.0.1"

	// ephemeralPort asks the kernel for an unused port.
	ephemeralPort = "0"
)

// errRedirectHost is returned when a redirect URL has no host.
var errRedirectHost = errors.New("redirect URL host is empty")

// Start listens for the OAuth redirect and returns a running server.
//
// An empty redirectURL binds 127.0.0.1 on an ephemeral port. A configured URL
// is parsed with [url.Parse]. An empty port is bound as port 0. After the
// bind, a hostname that differs from the configured hostname is restored and
// the configured path is kept. A bind failure is returned and does not exit
// the process.
//
// Parameters:
//   - ctx: cancels the bind. It does not stop a server that has started.
//   - redirectURL: provider redirect, or empty for an ephemeral loopback.
//
// Returns:
//   - *Server: the running callback server.
//   - error: non-nil when the URL is invalid or the bind fails.
func (factory *Factory) Start(
	ctx context.Context,
	redirectURL string,
) (*Server, error) {
	listener, resolved, err := listen(ctx, redirectURL)
	if err != nil {
		return nil, err
	}

	return newServer(listener, resolved, factory.Version), nil
}

// listen binds the loopback address for redirectURL.
//
// An empty redirectURL uses the ephemeral loopback listener. Any other value
// is treated as a configured redirect URL.
//
// Parameters:
//   - ctx: cancels the bind.
//   - redirectURL: provider redirect, or empty for an ephemeral loopback.
//
// Returns:
//   - [net.Listener]: bound listener. Nil when the bind fails.
//   - string: redirect URL using the bound port.
//   - error: non-nil when the URL is invalid or the bind fails.
func listen(
	ctx context.Context,
	redirectURL string,
) (net.Listener, string, error) {
	if redirectURL == "" {
		return listenLoopback(ctx)
	}

	return listenRedirect(ctx, redirectURL)
}

// listenLoopback binds 127.0.0.1 on an ephemeral port.
//
// Parameters:
//   - ctx: cancels the bind.
//
// Returns:
//   - [net.Listener]: bound loopback listener.
//   - string: http URL for the bound address.
//   - error: non-nil when the bind fails.
func listenLoopback(ctx context.Context) (net.Listener, string, error) {
	address := net.JoinHostPort(loopbackHost, ephemeralPort)

	listener, err := new(net.ListenConfig).Listen(ctx, networkTCP, address)
	if err != nil {
		return nil, "", fmt.Errorf("listen on loopback: %w", err)
	}

	return listener, loopbackURL(listener.Addr().String()), nil
}

// listenRedirect binds the host from a configured redirect URL.
//
// Parameters:
//   - ctx: cancels the bind.
//   - redirectURL: absolute or scheme-relative provider redirect.
//
// Returns:
//   - [net.Listener]: bound listener.
//   - string: configured URL with the bound port and original path.
//   - error: non-nil when the URL is invalid or the bind fails.
func listenRedirect(
	ctx context.Context,
	redirectURL string,
) (net.Listener, string, error) {
	parsed, err := url.Parse(redirectURL)
	if err != nil {
		return nil, "", fmt.Errorf("parse redirect URL: %w", err)
	}

	if parsed.Hostname() == "" {
		return nil, "", errRedirectHost
	}

	if parsed.Scheme == "" {
		parsed.Scheme = schemeHTTP
	}

	listener, err := listenHost(ctx, parsed)
	if err != nil {
		return nil, "", err
	}

	resolved, err := resolveRedirect(parsed, listener)
	if err != nil {
		return nil, "", closeListener(listener, err)
	}

	return listener, resolved, nil
}

// listenHost binds the configured hostname, using port 0 when none is set.
//
// Parameters:
//   - ctx: cancels the bind.
//   - parsed: redirect URL whose host and port select the bind address.
//
// Returns:
//   - [net.Listener]: bound listener.
//   - error: non-nil when the bind fails.
func listenHost(ctx context.Context, parsed *url.URL) (net.Listener, error) {
	port := parsed.Port()
	if port == "" {
		port = ephemeralPort
	}

	address := net.JoinHostPort(parsed.Hostname(), port)

	listener, err := new(net.ListenConfig).Listen(ctx, networkTCP, address)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", address, err)
	}

	return listener, nil
}

// resolveRedirect keeps the configured hostname and path with the bound port.
//
// If the kernel reports a different hostname, the configured hostname is
// restored. The path on parsed is left unchanged.
//
// Parameters:
//   - parsed: configured redirect URL. Its host is updated in place.
//   - listener: listener whose bound address supplies the port.
//
// Returns:
//   - string: resolved redirect URL.
//   - error: non-nil when the bound address cannot be split.
func resolveRedirect(parsed *url.URL, listener net.Listener) (string, error) {
	boundHost, boundPort, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return "", fmt.Errorf("split listener address: %w", err)
	}

	host := boundHost
	if boundHost != parsed.Hostname() {
		host = parsed.Hostname()
	}

	parsed.Host = net.JoinHostPort(host, boundPort)

	return parsed.String(), nil
}

// loopbackURL formats a bound address as an http URL without a path.
//
// Parameters:
//   - addr: host and port from the listener, including IPv6 brackets.
//
// Returns:
//   - string: http URL for addr.
func loopbackURL(addr string) string {
	parsed := new(url.URL)

	parsed.Scheme = schemeHTTP
	parsed.Host = addr

	return parsed.String()
}

// closeListener closes listener and joins that error with err.
//
// Parameters:
//   - listener: listener to close after a failed setup step.
//   - err: error that caused the listener to be discarded.
//
// Returns:
//   - error: err, plus a close error when the close fails.
func closeListener(listener net.Listener, err error) error {
	closeErr := listener.Close()
	if closeErr != nil {
		return errors.Join(err, fmt.Errorf("close listener: %w", closeErr))
	}

	return err
}
