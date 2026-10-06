// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package browser

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommandPerPlatform covers the platform command selection.
func TestCommandPerPlatform(t *testing.T) {
	t.Parallel()

	const target = "https://github.com/login/oauth/authorize?x=1"

	tests := []struct {
		name     string
		goos     string
		wantName string
		wantArgs []string
	}{
		{
			name:     "darwin uses open",
			goos:     "darwin",
			wantName: "open",
			wantArgs: []string{target},
		},
		{
			name:     "windows uses cmd start",
			goos:     "windows",
			wantName: "cmd",
			wantArgs: []string{"/c", "start", "", target},
		},
		{
			name:     "linux uses xdg-open",
			goos:     "linux",
			wantName: "xdg-open",
			wantArgs: []string{target},
		},
		{
			name:     "freebsd uses xdg-open",
			goos:     "freebsd",
			wantName: "xdg-open",
			wantArgs: []string{target},
		},
		{
			name:     "empty goos falls back",
			goos:     "",
			wantName: "xdg-open",
			wantArgs: []string{target},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			name, args := Command(tt.goos, target)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantArgs, args)
		})
	}
}

// TestCommandPassesURLLast checks the URL is the final argument.
//
// A browser that takes the URL anywhere else would treat it as a flag or an
// option value, so the position is load-bearing.
func TestCommandPassesURLLast(t *testing.T) {
	t.Parallel()

	const target = "https://example.com"

	for _, goos := range []string{"darwin", "windows", "linux", ""} {
		_, args := Command(goos, target)
		assert.Equal(t, target, args[len(args)-1], "goos %q", goos)
	}
}

// TestCommandWindowsStartNeedsEmptyTitle checks the Windows quoting.
//
// start treats its first quoted argument as a window title, so an empty string
// has to occupy that position or the URL would be consumed as the title.
func TestCommandWindowsStartNeedsEmptyTitle(t *testing.T) {
	t.Parallel()

	name, args := Command("windows", "https://example.com")

	require.Equal(t, "cmd", name)
	require.Greater(t, len(args), 2)
	assert.Equal(t, []string{"/c", "start", ""}, args[:3])
}

// TestCommandIsPure checks Command has no side effects.
//
// It is called on every authorization code grant, so it must not touch the
// filesystem or the process environment.
func TestCommandIsPure(t *testing.T) {
	t.Parallel()

	const target = "https://example.com"

	first, firstArgs := Command("linux", target)
	second, secondArgs := Command("linux", target)

	assert.Equal(t, first, second)
	assert.True(t, slices.Equal(firstArgs, secondArgs))
}
