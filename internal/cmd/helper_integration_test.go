// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/mock"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd"
	cmdget "github.com/nicholas-fedor/git-credential-oauth/internal/cmd/get"
	mockGet "github.com/nicholas-fedor/git-credential-oauth/internal/cmd/get/mocks"
	mockHelper "github.com/nicholas-fedor/git-credential-oauth/internal/cmd/helper/mocks"
	"github.com/nicholas-fedor/git-credential-oauth/internal/helper"
)

// TestConfigurePassesStorageThrough checks the flag reaches the helper.
func TestConfigurePassesStorageThrough(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		storage string
		want    helper.Storage
		wantErr bool
	}{
		{name: "auto", storage: "auto", want: helper.StorageAuto},
		{name: "libsecret", storage: "libsecret", want: helper.StorageLibsecret},
		{name: "osxkeychain", storage: "osxkeychain", want: helper.StorageOSXKeychain},
		{name: "wincred", storage: "wincred", want: helper.StorageWincred},
		{name: "cache", storage: "cache", want: helper.StorageCache},
		{name: "store", storage: "store", want: helper.StorageStore},
		{name: "none", storage: "none", want: helper.StorageNone},
		{name: "unknown is rejected", storage: "1password", wantErr: true},
		{name: "empty is rejected", storage: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			helperMock := mockHelper.NewMockHelper(t)

			if !tt.wantErr {
				helperMock.EXPECT().
					Configure(mock.Anything, mock.Anything).
					Return(nil).
					Once()
			}

			root := newRootWithHelper(
				t, mockGet.NewMockGetter(t), "",
				nil, io.Discard, io.Discard, helperMock,
			)
			root.SetArgs([]string{"configure", "--storage", tt.storage})

			err := root.Execute()
			if tt.wantErr {
				if err == nil {
					t.Fatal("Execute() error = nil, want a storage error")
				}

				return
			}

			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			helperMock.AssertCalled(t, "Configure", mock.Anything, mock.MatchedBy(
				func(opts helper.Options) bool { return opts.Storage == tt.want },
			))
		})
	}
}

// TestConfigureDefaultsToAuto checks the flag default when none is given.
func TestConfigureDefaultsToAuto(t *testing.T) {
	t.Parallel()

	helperMock := mockHelper.NewMockHelper(t)
	helperMock.EXPECT().Configure(mock.Anything, mock.Anything).Return(nil).Once()

	root := newRootWithHelper(t, mockGet.NewMockGetter(t), "",
		nil, io.Discard, io.Discard, helperMock)
	root.SetArgs([]string{"configure"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	helperMock.AssertCalled(t, "Configure", mock.Anything, mock.MatchedBy(
		func(opts helper.Options) bool { return opts.Storage == helper.StorageAuto },
	))
}

// TestConfigureRejectsUnknownStorage covers the validation path.
func TestConfigureRejectsUnknownStorage(t *testing.T) {
	t.Parallel()

	helperMock := mockHelper.NewMockHelper(t)

	root := newRootWithHelper(
		t, mockGet.NewMockGetter(t), "",
		nil, io.Discard, io.Discard, helperMock,
	)
	root.SetArgs([]string{"configure", "--storage", "not-a-helper"})

	if err := root.Execute(); err == nil {
		t.Fatal("Execute() error = nil, want a storage error")
	}

	helperMock.AssertNotCalled(t, "Configure", mock.Anything, mock.Anything)
}

// TestConfigureWritesSuccessToStderr checks the success line stream.
//
// git reads this program's standard output as a credential, so a success
// message must go to standard error.
func TestConfigureWritesSuccessToStderr(t *testing.T) {
	t.Parallel()

	helperMock := mockHelper.NewMockHelper(t)
	helperMock.EXPECT().Configure(mock.Anything, mock.Anything).Return(nil).Once()

	var stdout, stderr bytes.Buffer

	root := newRootWithHelper(
		t, mockGet.NewMockGetter(t), "",
		nil, &stdout, &stderr, helperMock,
	)
	root.SetArgs([]string{"configure"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	if !strings.Contains(stderr.String(), "configured") {
		t.Fatalf("stderr = %q, want a configured message", stderr.String())
	}
}

// TestConfigureSurfacesHelperFailure checks a git failure is not swallowed.
func TestConfigureSurfacesHelperFailure(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")
	helperMock := mockHelper.NewMockHelper(t)
	helperMock.EXPECT().
		Configure(mock.Anything, mock.Anything).
		Return(boom).
		Once()

	root := newRootWithHelper(
		t, mockGet.NewMockGetter(t), "",
		nil, io.Discard, io.Discard, helperMock,
	)
	root.SetArgs([]string{"configure"})

	err := root.Execute()
	if !errors.Is(err, boom) {
		t.Fatalf("Execute() error = %v, want %v", err, boom)
	}
}

// TestUnconfigureSurfacesHelperFailure checks a removal failure propagates.
func TestUnconfigureSurfacesHelperFailure(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")
	helperMock := mockHelper.NewMockHelper(t)
	helperMock.EXPECT().
		Unconfigure(mock.Anything, mock.Anything).
		Return(boom).
		Once()

	root := newRootWithHelper(
		t, mockGet.NewMockGetter(t), "",
		nil, io.Discard, io.Discard, helperMock,
	)
	root.SetArgs([]string{"unconfigure"})

	err := root.Execute()
	if !errors.Is(err, boom) {
		t.Fatalf("Execute() error = %v, want %v", err, boom)
	}
}

// TestUnconfigureIgnoresStorageFlag checks the removal plan is storage-agnostic.
//
// Unconfigure removes every known helper value, so the storage flag is
// accepted but must not change what is removed. The flag is only bound on
// configure, so passing it to unconfigure is itself a usage error.
func TestUnconfigureRejectsStorageFlag(t *testing.T) {
	t.Parallel()

	helperMock := mockHelper.NewMockHelper(t)

	root := newRootWithHelper(
		t, mockGet.NewMockGetter(t), "",
		nil, io.Discard, io.Discard, helperMock,
	)
	root.SetArgs([]string{"unconfigure", "--storage", "store"})

	if err := root.Execute(); err == nil {
		t.Fatal("Execute() error = nil, want an unknown flag error")
	}

	helperMock.AssertNotCalled(t, "Unconfigure", mock.Anything, mock.Anything)
}

// newRootWithHelper builds the command tree with an explicit helper mock.
//
// Parameters:
//   - t: test handle.
//   - getter: credential getter.
//   - info: version metadata.
//   - stdin: request input.
//   - stdout: response output.
//   - stderr: diagnostic output.
//   - helperMock: helper service.
//
// Returns:
//   - *cobra.Command: root command ready to execute.
func newRootWithHelper(
	t *testing.T,
	getter cmdget.Getter,
	info string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	helperMock *mockHelper.MockHelper,
) *cobra.Command {
	t.Helper()

	return cmd.NewRoot(t.Context(), cmd.Dependencies{
		Get:     getter,
		Helper:  helperMock,
		Version: info,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
	})
}
