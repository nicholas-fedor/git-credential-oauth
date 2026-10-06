// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package callback

import (
	"errors"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAddr is a net.Addr with a fixed string form.
type fakeAddr string

func (a fakeAddr) Network() string { return networkTCP }
func (a fakeAddr) String() string  { return string(a) }

// fakeListener is a listener that never accepts and reports a chosen address.
type fakeListener struct {
	closeErr error
	addr     fakeAddr
	closed   bool
}

func (l *fakeListener) Accept() (net.Conn, error) { return nil, net.ErrClosed }
func (l *fakeListener) Addr() net.Addr            { return l.addr }

// Close records the call and returns the configured error.
func (l *fakeListener) Close() error {
	l.closed = true

	return l.closeErr
}

var errCloseFailed = errors.New("close failed")

// TestListenWithoutARedirectBindsLoopback covers the default.
//
// RFC 8252 section 7.3 has a native client listen on the loopback address on a
// port the system picks, so no fixed port can be taken first by another
// program.
func TestListenWithoutARedirectBindsLoopback(t *testing.T) {
	t.Parallel()

	listener, resolved, err := listen(t.Context(), "")
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	assert.Regexp(t, `^http://127\.0\.0\.1:\d+$`, resolved)
	assert.Equal(t, listener.Addr().String(), resolved[len("http://"):])
}

// TestListenRedirectKeepsThePathAndAddsTheBoundPort covers a configured URL.
func TestListenRedirectKeepsThePathAndAddsTheBoundPort(t *testing.T) {
	t.Parallel()

	listener, resolved, err := listenRedirect(t.Context(), "http://127.0.0.1/cb?x=1")
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	parsed, err := url.Parse(resolved)
	require.NoError(t, err)

	assert.Equal(t, "http", parsed.Scheme)
	assert.Equal(t, "127.0.0.1", parsed.Hostname())
	assert.NotEqual(t, "0", parsed.Port(), "the bound port replaces the ephemeral one")
	assert.Equal(t, "/cb", parsed.Path)
	assert.Equal(t, "x=1", parsed.RawQuery)
}

// TestListenRedirectDefaultsTheScheme covers a value without one.
func TestListenRedirectDefaultsTheScheme(t *testing.T) {
	t.Parallel()

	listener, resolved, err := listenRedirect(t.Context(), "//127.0.0.1/cb")
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	assert.Regexp(t, `^http://127\.0\.0\.1:\d+/cb$`, resolved)
}

// TestListenRedirectRejectsBadValues covers each refusal.
func TestListenRedirectRejectsBadValues(t *testing.T) {
	t.Parallel()

	_, _, err := listenRedirect(t.Context(), "http://127.0.0.1:%zz/")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse redirect URL")

	_, _, err = listenRedirect(t.Context(), "/callback")
	require.ErrorIs(t, err, errRedirectHost)

	_, _, err = listenRedirect(t.Context(), "http://127.0.0.1:99999/")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listen on 127.0.0.1:99999")
}

// TestListenRedirectReportsAPortInUse covers a fixed port already taken.
//
// A fixed redirect port can be claimed by another program first, which is why
// the default is an ephemeral one. When that happens the request must fail
// rather than wait on a port it does not own.
func TestListenRedirectReportsAPortInUse(t *testing.T) {
	t.Parallel()

	taken, err := new(net.ListenConfig).Listen(t.Context(), networkTCP, "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() { _ = taken.Close() })

	_, _, err = listenRedirect(t.Context(), "http://"+taken.Addr().String()+"/cb")
	require.Error(t, err)
}

// TestResolveRedirectKeepsTheConfiguredHostname covers a named loopback host.
func TestResolveRedirectKeepsTheConfiguredHostname(t *testing.T) {
	t.Parallel()

	parsed, err := url.Parse("http://localhost/cb")
	require.NoError(t, err)

	got, err := resolveRedirect(parsed, &fakeListener{addr: "127.0.0.1:4242"})
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:4242/cb", got)
}

// TestResolveRedirectReportsAnUnreadableAddress covers a listener without a port.
func TestResolveRedirectReportsAnUnreadableAddress(t *testing.T) {
	t.Parallel()

	parsed, err := url.Parse("http://127.0.0.1/cb")
	require.NoError(t, err)

	_, err = resolveRedirect(parsed, &fakeListener{addr: "no-port"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "split listener address")
}

// TestCloseListenerJoinsBothErrors keeps the original failure visible.
func TestCloseListenerJoinsBothErrors(t *testing.T) {
	t.Parallel()

	quiet := &fakeListener{}
	require.ErrorIs(t, closeListener(quiet, errRedirectHost), errRedirectHost)
	assert.True(t, quiet.closed)

	failing := &fakeListener{closeErr: errCloseFailed}
	err := closeListener(failing, errRedirectHost)
	require.ErrorIs(t, err, errRedirectHost)
	require.ErrorIs(t, err, errCloseFailed)
}

// TestLoopbackURLHasNoPath checks the default redirect form.
func TestLoopbackURLHasNoPath(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "http://127.0.0.1:4242", loopbackURL("127.0.0.1:4242"))
	assert.Equal(t, "http://[::1]:4242", loopbackURL("[::1]:4242"))
}
