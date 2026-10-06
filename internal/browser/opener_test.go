// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package browser

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errLookPath is returned by a stubbed resolver.
var errLookPath = errors.New("not on PATH")

// errRun is returned by a stubbed starter.
var errRun = errors.New("run failed")

// TestOpenUsesConfiguredPlatform checks GOOS selects the command.
func TestOpenUsesConfiguredPlatform(t *testing.T) {
	t.Parallel()

	var gotName string

	var gotArgs []string

	opener := &Opener{
		LookPath: func(name string) (string, error) {
			gotName = name
			return "/usr/bin/" + name, nil
		},
		Run: func(_ context.Context, name string, args ...string) error {
			gotArgs = args

			return nil
		},
		GOOS: "darwin",
	}

	require.NoError(t, opener.Open(t.Context(), "https://example.com"))
	assert.Equal(t, "open", gotName)
	assert.Equal(t, []string{"https://example.com"}, gotArgs)
}

// TestOpenResolvesBeforeRunning checks a missing command is not run.
//
// xdg-open on a headless host does not exist, and starting it would report a
// confusing start failure instead of a missing browser.
func TestOpenResolvesBeforeRunning(t *testing.T) {
	t.Parallel()

	run := false

	opener := &Opener{
		LookPath: func(string) (string, error) { return "", errLookPath },
		Run: func(context.Context, string, ...string) error {
			run = true

			return nil
		},
		GOOS: "linux",
	}

	err := opener.Open(t.Context(), "https://example.com")
	require.ErrorIs(t, err, ErrNoBrowser)
	require.ErrorIs(t, err, errLookPath, "the lookup failure is preserved")
	assert.False(t, run, "a missing command is not started")
}

// TestOpenSurfacesRunFailure checks a start failure is distinct.
func TestOpenSurfacesRunFailure(t *testing.T) {
	t.Parallel()

	opener := &Opener{
		LookPath: func(name string) (string, error) { return name, nil },
		Run:      func(context.Context, string, ...string) error { return errRun },
		GOOS:     "linux",
	}

	err := opener.Open(t.Context(), "https://example.com")
	require.ErrorIs(t, err, errRun)
	assert.NotErrorIs(t, err, ErrNoBrowser, "a start failure is not a missing command")
}

// TestOpenPassesResolvedNameNotPath checks the command name is executed.
//
// LookPath is consulted only to confirm the command exists, so passing its
// resolved path would break a shell wrapper that expects to be found on PATH.
func TestOpenPassesResolvedNameNotPath(t *testing.T) {
	t.Parallel()

	var ran string

	opener := &Opener{
		LookPath: func(string) (string, error) { return "/opt/bin/xdg-open", nil },
		Run: func(_ context.Context, name string, _ ...string) error {
			ran = name

			return nil
		},
		GOOS: "linux",
	}

	require.NoError(t, opener.Open(t.Context(), "https://example.com"))
	assert.Equal(t, "xdg-open", ran)
}

// TestOpenDefaultsToProcessGOOS checks an empty GOOS falls back.
//
// The fallback is the platform the test runs on, so the expected command is
// computed from runtime.GOOS rather than written out.
func TestOpenDefaultsToProcessGOOS(t *testing.T) {
	t.Parallel()

	var gotName string

	opener := &Opener{
		LookPath: func(name string) (string, error) {
			gotName = name

			return name, nil
		},
		Run:  func(context.Context, string, ...string) error { return nil },
		GOOS: "",
	}

	require.NoError(t, opener.Open(t.Context(), "https://example.com"))

	want, _ := Command(runtime.GOOS, "https://example.com")
	assert.Equal(t, want, gotName)
}

// TestOpenRespectsCancellation checks a canceled context stops the browser.
func TestOpenRespectsCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	opener := &Opener{
		LookPath: func(name string) (string, error) { return name, nil },
		Run: func(runCtx context.Context, _ string, _ ...string) error {
			return runCtx.Err()
		},
		GOOS: "linux",
	}

	require.Error(t, opener.Open(ctx, "https://example.com"))
}

// TestNilFieldsUseDefaults checks a partially configured Opener is usable.
func TestNilFieldsUseDefaults(t *testing.T) {
	t.Parallel()

	opener := &Opener{}
	assert.NotNil(t, opener.lookPath())
	assert.NotNil(t, opener.run())
}
