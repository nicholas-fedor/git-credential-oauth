// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/oauth"
)

// TestCallbackFactoryStartReturnsCallback checks the adapter bridges the
// concrete server to the consumer interface.
func TestCallbackFactoryStartReturnsCallback(t *testing.T) {
	t.Parallel()

	factory := newCallbackFactory()

	server, err := factory.Start(t.Context(), "")
	require.NoError(t, err)
	require.NotNil(t, server)

	t.Cleanup(func() {
		_ = server.Close()
	})

	assert.Contains(t, server.RedirectURL(), "127.0.0.1")
}

// TestCallbackFactorySatisfiesInterface is a compile-time style check.
func TestCallbackFactorySatisfiesInterface(t *testing.T) {
	t.Parallel()

	var factory oauth.CallbackFactory = newCallbackFactory()
	assert.NotNil(t, factory)
}

// TestCallbackFactoryCloseIsIdempotent checks the server closes cleanly once.
//
// The authorization-code grant closes the callback on every path, including
// error paths, so a second close must not panic.
func TestCallbackFactoryCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	server, err := newCallbackFactory().Start(t.Context(), "")
	require.NoError(t, err)
	require.NotNil(t, server)

	require.NoError(t, server.Close())
	assert.NotPanics(t, func() {
		_ = server.Close()
	})
}

// TestCallbackFactoryStartIsNotAffectedByUnusedContext documents that the
// factory takes a context it passes straight through.
func TestCallbackFactoryStartIsNotAffectedByUnusedContext(t *testing.T) {
	t.Parallel()

	server, err := newCallbackFactory().Start(t.Context(), "")
	require.NoError(t, err)
	require.NotNil(t, server)

	require.NoError(t, server.Close())
	assert.True(t, strings.HasPrefix(server.RedirectURL(), "http://127.0.0.1:"))
}

// TestNewCallbackFactoryUsesVersion checks the success page carries the build
// version, which is what a user reports when something goes wrong.
func TestNewCallbackFactoryUsesVersion(t *testing.T) {
	t.Parallel()

	factory := newCallbackFactory()
	require.NotNil(t, factory.inner)

	concrete := factory.inner
	assert.NotEmpty(t, concrete.Version)
}

// TestCallbackFactoryStartReturnsANilInterfaceOnFailure checks a failed start
// is visible.
//
// Returning a typed nil server inside the interface would compare unequal to
// nil, and the grant would call methods on it.
func TestCallbackFactoryStartReturnsANilInterfaceOnFailure(t *testing.T) {
	t.Parallel()

	server, err := newCallbackFactory().Start(t.Context(), "/no-host")
	require.ErrorIs(t, err, errStartCallback)
	assert.Nil(t, server)
}
