// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package options

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/helper"
)

// failWriter fails every write so the error paths are reachable.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("write output: disk full")
}

// TestNew checks the pre-parse defaults.
//
// Device and bearer must default off so an existing browser-flow configuration
// is not changed by an upgrade.
func TestNew(t *testing.T) {
	t.Parallel()

	opts := New()

	require.NotNil(t, opts)
	assert.False(t, opts.Device)
	assert.False(t, opts.Bearer)
	assert.False(t, opts.Verbose)
	assert.Equal(t, string(helper.StorageAuto), opts.Storage)
}

// TestBindPersistent checks every shared flag reaches its destination.
//
// git appends helper-wide arguments to whichever helper it invokes, so a flag
// bound on root has to survive parsing under every subcommand.
func TestBindPersistent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want Options
	}{
		{name: "defaults", args: nil},
		{
			name: "long flags",
			args: []string{"--verbose", "--device", "--bearer"},
			want: Options{Device: true, Bearer: true, Verbose: true},
		},
		{
			name: "short verbose",
			args: []string{"-v"},
			want: Options{Verbose: true},
		},
		{
			name: "explicit false",
			args: []string{"--verbose=false", "--device=false", "--bearer=false"},
			want: Options{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := New()
			root := &cobra.Command{Use: "root"}
			BindPersistent(root, opts)

			err := root.PersistentFlags().Parse(tt.args)
			require.NoError(t, err)

			assert.Equal(t, tt.want.Device, opts.Device, "device")
			assert.Equal(t, tt.want.Bearer, opts.Bearer, "bearer")
			assert.Equal(t, tt.want.Verbose, opts.Verbose, "verbose")
		})
	}
}

// TestBindStorage checks the storage flag and its documented value set.
//
// The usage string lists the accepted helpers, so the flag help and the
// validator are asserted against the same source.
func TestBindStorage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		args []string
	}{
		{name: "defaults to auto", args: nil, want: string(helper.StorageAuto)},
		{name: "explicit value", args: []string{"--storage", "libsecret"}, want: "libsecret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := New()
			command := &cobra.Command{Use: "configure"}
			BindStorage(command, opts)

			err := command.Flags().Parse(tt.args)
			require.NoError(t, err)

			assert.Equal(t, tt.want, opts.Storage)
		})
	}

	t.Run("usage lists every accepted helper", func(t *testing.T) {
		t.Parallel()

		opts := New()
		command := &cobra.Command{Use: "configure"}
		BindStorage(command, opts)

		flag := command.Flags().Lookup(flagStorage)
		require.NotNil(t, flag)

		for _, value := range helper.Values() {
			assert.Contains(t, flag.Usage, value)
		}
	})
}

// TestParseStorage covers acceptance and rejection.
func TestParseStorage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    helper.Storage
		wantErr bool
	}{
		{name: "auto", value: "auto", want: helper.StorageAuto},
		{name: "libsecret", value: "libsecret", want: helper.StorageLibsecret},
		{name: "osxkeychain", value: "osxkeychain", want: helper.StorageOSXKeychain},
		{name: "wincred", value: "wincred", want: helper.StorageWincred},
		{name: "cache", value: "cache", want: helper.StorageCache},
		{name: "store", value: "store", want: helper.StorageStore},
		{name: "none", value: "none", want: helper.StorageNone},
		{name: "unknown is rejected", value: "1password", wantErr: true},
		{name: "empty is rejected", value: "", wantErr: true},
		{
			name:    "case matters",
			value:   "AUTO",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseStorage(tt.value)

			if tt.wantErr {
				require.ErrorIs(t, err, errUnsupportedStorage)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
