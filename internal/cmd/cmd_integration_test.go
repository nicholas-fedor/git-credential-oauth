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
)

func TestNewRootCapability(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := execute(t, []string{"capability"}, "unused")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if stdout != "version 0\ncapability authtype\n" {
		t.Fatalf("stdout = %q", stdout)
	}

	if stderr != "" {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestNewRootVersion(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := execute(t, []string{"version"}, "9.9.9")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if stdout != "git-credential-oauth 9.9.9\n" {
		t.Fatalf("stdout = %q", stdout)
	}

	if stderr != "" {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestNewRootGetMissingStdin(t *testing.T) {
	t.Parallel()

	getter := mockGet.NewMockGetter(t)
	var stdout, stderr bytes.Buffer

	root := newRoot(t, getter, "", failReader{}, &stdout, &stderr)
	root.SetArgs([]string{"get"})

	err := root.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want a read error")
	}

	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Execute() error = %v, want %v", err, io.ErrUnexpectedEOF)
	}

	getter.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
}

func TestNewRootSilence(t *testing.T) {
	t.Parallel()

	root := newRoot(t, mockGet.NewMockGetter(t), "", failReader{}, io.Discard, io.Discard)
	if !root.SilenceUsage {
		t.Fatal("SilenceUsage = false")
	}

	if !root.SilenceErrors {
		t.Fatal("SilenceErrors = false")
	}
}

// TestNewRootStoreEraseAreSilent covers the operations git calls but this
// program does not implement.
//
// gitcredentials(7) requires an unrecognized operation to be ignored
// silently, and git inherits this program's standard error, so any output here
// would appear in the user's terminal on every authenticated push.
func TestNewRootStoreEraseAreSilent(t *testing.T) {
	t.Parallel()

	for _, operation := range []string{"store", "erase"} {
		t.Run(operation, func(t *testing.T) {
			t.Parallel()

			stdout, stderr, err := execute(t, []string{operation}, "")
			if err != nil {
				t.Fatalf("Execute() error = %v, want nil", err)
			}

			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}

			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
		})
	}
}

// TestNewRootUnknownOperationStillFails checks the added commands did not
// swallow genuine mistakes.
//
// Only the operations git actually sends are ignored; a typo must still
// surface as an error.
func TestNewRootUnknownOperationStillFails(t *testing.T) {
	t.Parallel()

	_, _, err := execute(t, []string{"aprove"}, "")
	if err == nil {
		t.Fatal("Execute() error = nil, want an unknown command error")
	}
}

// TestNewRootHelpIsWrittenWhenNoSubcommandGiven covers the bare invocation.
func TestNewRootHelpIsWrittenWhenNoSubcommandGiven(t *testing.T) {
	t.Parallel()

	stdout, _, err := execute(t, nil, "")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout, "Generate credential") {
		t.Fatalf("stdout does not list the get command: %q", stdout)
	}
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func execute(t *testing.T, args []string, info string) (string, string, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer

	root := newRoot(t, mockGet.NewMockGetter(t), info, failReader{}, &stdout, &stderr)
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)

	err := root.Execute()

	return stdout.String(), stderr.String(), err
}

func newRoot(
	t *testing.T,
	getter cmdget.Getter,
	info string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) *cobra.Command {
	t.Helper()

	return cmd.NewRoot(t.Context(), cmd.Dependencies{
		Get:     getter,
		Helper:  mockHelper.NewMockHelper(t),
		Version: info,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
	})
}
