// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package helper_test

import (
	"bytes"
	"errors"
	"io"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cmdhelper "github.com/nicholas-fedor/git-credential-oauth/internal/cmd/helper"
	mockHelper "github.com/nicholas-fedor/git-credential-oauth/internal/cmd/helper/mocks"
	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/options"
	"github.com/nicholas-fedor/git-credential-oauth/internal/helper"
)

// failWriter fails every write so the diagnostic error path is reachable.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("write output: closed pipe")
}

// TestConfigureMetadata checks the command identity and flag surface.
//
// The storage flag is bound on configure only, because unconfigure removes
// every helper value and has nothing to select.
func TestConfigureMetadata(t *testing.T) {
	t.Parallel()

	command := cmdhelper.NewConfigureCommand(
		t.Context(),
		mockHelper.NewMockHelper(t),
		io.Discard,
		options.New(),
	)

	assert.Equal(t, "configure", command.Name())
	assert.NotNil(t, command.Flags().Lookup("storage"))
	require.NoError(t, command.Args(command, nil))
	require.Error(t, command.Args(command, []string{"extra"}))
}

// TestUnconfigureMetadata checks the command identity and flag surface.
//
// Accepting --storage here would suggest the removal plan depends on it.
func TestUnconfigureMetadata(t *testing.T) {
	t.Parallel()

	command := cmdhelper.NewUnconfigureCommand(
		t.Context(),
		mockHelper.NewMockHelper(t),
		io.Discard,
		options.New(),
	)

	assert.Equal(t, "unconfigure", command.Name())
	assert.Nil(t, command.Flags().Lookup("storage"))
	require.NoError(t, command.Args(command, nil))
	require.Error(t, command.Args(command, []string{"extra"}))
}

// TestConfigurePassesStorageAndDeviceThrough checks the flags reach the
// service alongside the running operating system.
//
// The device flag is persistent on the root, so it arrives through the options
// rather than the configure flag set.
func TestConfigurePassesStorageAndDeviceThrough(t *testing.T) {
	t.Parallel()

	tests := []struct {
		opts    *options.Options
		name    string
		storage helper.Storage
		args    []string
		device  bool
	}{
		{
			name:    "defaults to auto",
			args:    []string{},
			opts:    options.New(),
			storage: helper.StorageAuto,
		},
		{
			name:    "explicit storage",
			args:    []string{"--storage", "libsecret"},
			opts:    options.New(),
			storage: helper.StorageLibsecret,
		},
		{
			name:    "device flow is recorded",
			args:    []string{"--storage", "wincred"},
			opts:    &options.Options{Storage: "wincred", Device: true},
			storage: helper.StorageWincred,
			device:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := mockHelper.NewMockHelper(t)
			service.EXPECT().
				Configure(mock.Anything, wantConfigure(tt.storage, tt.device)).
				Return(nil).
				Once()

			var stderr bytes.Buffer

			command := cmdhelper.NewConfigureCommand(t.Context(), service, &stderr, tt.opts)
			command.SetArgs(tt.args)

			require.NoError(t, command.ExecuteContext(t.Context()))
			assert.Contains(t, stderr.String(), "configured")
		})
	}
}

// TestConfigureRejectsUnknownStorage checks validation happens before git is
// touched, so a typo cannot half-install a helper.
func TestConfigureRejectsUnknownStorage(t *testing.T) {
	t.Parallel()

	service := mockHelper.NewMockHelper(t)

	command := cmdhelper.NewConfigureCommand(
		t.Context(),
		service,
		io.Discard,
		options.New(),
	)
	command.SetArgs([]string{"--storage", "not-a-helper"})

	require.Error(t, command.ExecuteContext(t.Context()))
	service.AssertNotCalled(t, "Configure", mock.Anything, mock.Anything)
}

// TestConfigureFailures checks that a git failure and an unwritable
// diagnostic are both surfaced.
func TestConfigureFailures(t *testing.T) {
	t.Parallel()

	t.Run("a service failure is not swallowed", func(t *testing.T) {
		t.Parallel()

		boom := errors.New("git config failed")

		service := mockHelper.NewMockHelper(t)
		service.EXPECT().
			Configure(mock.Anything, mock.Anything).
			Return(boom).
			Once()

		command := cmdhelper.NewConfigureCommand(
			t.Context(),
			service,
			io.Discard,
			options.New(),
		)
		command.SetArgs([]string{})

		require.ErrorIs(t, command.ExecuteContext(t.Context()), boom)
	})

	t.Run("an unwritable diagnostic is reported", func(t *testing.T) {
		t.Parallel()

		service := mockHelper.NewMockHelper(t)
		service.EXPECT().Configure(mock.Anything, mock.Anything).Return(nil).Once()

		command := cmdhelper.NewConfigureCommand(
			t.Context(),
			service,
			failWriter{},
			options.New(),
		)
		command.SetArgs([]string{})

		require.Error(t, command.ExecuteContext(t.Context()))
	})
}

// TestUnconfigureRemovesEveryHelperValue checks the removal plan ignores the
// installed storage backend.
func TestUnconfigureRemovesEveryHelperValue(t *testing.T) {
	t.Parallel()

	service := mockHelper.NewMockHelper(t)
	service.EXPECT().
		Unconfigure(mock.Anything, wantConfigure(helper.StorageAuto, true)).
		Return(nil).
		Once()

	var stderr bytes.Buffer

	command := cmdhelper.NewUnconfigureCommand(
		t.Context(),
		service,
		&stderr,
		&options.Options{Device: true},
	)
	command.SetArgs([]string{})

	require.NoError(t, command.ExecuteContext(t.Context()))
	assert.Contains(t, stderr.String(), "unconfigured")
}

// TestUnconfigureFailures checks that a removal failure and an unwritable
// diagnostic are both surfaced.
func TestUnconfigureFailures(t *testing.T) {
	t.Parallel()

	t.Run("a service failure is not swallowed", func(t *testing.T) {
		t.Parallel()

		boom := errors.New("git config failed")

		service := mockHelper.NewMockHelper(t)
		service.EXPECT().
			Unconfigure(mock.Anything, mock.Anything).
			Return(boom).
			Once()

		command := cmdhelper.NewUnconfigureCommand(
			t.Context(),
			service,
			io.Discard,
			options.New(),
		)
		command.SetArgs([]string{})

		require.ErrorIs(t, command.ExecuteContext(t.Context()), boom)
	})

	t.Run("an unwritable diagnostic is reported", func(t *testing.T) {
		t.Parallel()

		service := mockHelper.NewMockHelper(t)
		service.EXPECT().Unconfigure(mock.Anything, mock.Anything).Return(nil).Once()

		command := cmdhelper.NewUnconfigureCommand(
			t.Context(),
			service,
			failWriter{},
			options.New(),
		)
		command.SetArgs([]string{})

		require.Error(t, command.ExecuteContext(t.Context()))
	})
}

// wantConfigure matches the options both commands build for the service.
func wantConfigure(storage helper.Storage, device bool) any {
	return mock.MatchedBy(func(opts helper.Options) bool {
		return opts.Storage == storage && opts.Device == device && opts.GOOS == runtime.GOOS
	})
}

// compile-time check that the generated mock satisfies the interface.
var _ cmdhelper.Helper = (*mockHelper.MockHelper)(nil)
