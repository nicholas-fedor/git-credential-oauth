// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd_test

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd"
	mockGet "github.com/nicholas-fedor/git-credential-oauth/internal/cmd/get/mocks"
	mockHelper "github.com/nicholas-fedor/git-credential-oauth/internal/cmd/helper/mocks"
	mockCmd "github.com/nicholas-fedor/git-credential-oauth/internal/cmd/mocks"
)

// TestVerboseFlagReachesGate checks the flag survives cobra parsing.
//
// The composition root wires its logger before Execute runs, so the only way it
// learns the flag value is this callback. A silent gate would leave debug
// logging permanently off.
func TestVerboseFlagReachesGate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "long form opens the gate", args: []string{"--verbose", "capability"}, want: true},
		{name: "short form opens the gate", args: []string{"-v", "capability"}, want: true},
		{name: "no flag leaves it closed", args: []string{"capability"}, want: false},
		{
			name: "explicit false leaves it closed",
			args: []string{"--verbose=false", "capability"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gate := mockCmd.NewMockDebugGate(t)
			gate.EXPECT().Set(tt.want).Once()

			var stdout, stderr bytes.Buffer

			root := cmd.NewRoot(t.Context(), cmd.Dependencies{
				Get:     mockGet.NewMockGetter(t),
				Helper:  mockHelper.NewMockHelper(t),
				Version: "unused",
				Stdin:   failReader{},
				Stdout:  &stdout,
				Stderr:  &stderr,
				Debug:   gate,
			})
			root.SetArgs(tt.args)

			require.NoError(t, root.Execute())
			gate.AssertExpectations(t)
		})
	}
}

// TestGateIsToleratedWhenAbsent checks a tree without a gate still runs.
//
// The gate is optional so the command package does not force a logger on every
// caller, including tests.
func TestGateIsToleratedWhenAbsent(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	root := cmd.NewRoot(t.Context(), cmd.Dependencies{
		Get:     mockGet.NewMockGetter(t),
		Helper:  mockHelper.NewMockHelper(t),
		Version: "unused",
		Stdin:   failReader{},
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	root.SetArgs([]string{"--verbose", "capability"})

	require.NoError(t, root.Execute())
	assert.Contains(t, stdout.String(), "capability authtype")
}

// TestVerboseIsAPersistentFlag checks every subcommand accepts it.
//
// git appends helper-wide arguments to whichever helper it invokes, so a flag
// that only the root accepts would break an otherwise valid invocation.
func TestVerboseIsAPersistentFlag(t *testing.T) {
	t.Parallel()

	for _, operation := range []string{"capability", "version", "store", "erase"} {
		t.Run(operation, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer

			root := cmd.NewRoot(t.Context(), cmd.Dependencies{
				Get:     mockGet.NewMockGetter(t),
				Helper:  mockHelper.NewMockHelper(t),
				Version: "unused",
				Stdin:   failReader{},
				Stdout:  &stdout,
				Stderr:  &stderr,
			})
			root.SetArgs([]string{"--verbose", operation})

			assert.NoError(t, root.Execute(), "operation %q", operation)
		})
	}
}

// compile-time check that the gate mock satisfies the interface it stands in
// for.
var _ cmd.DebugGate = (*mockCmd.MockDebugGate)(nil)

// compile-time check that cobra still owns the root type.
var _ *cobra.Command = (*cobra.Command)(nil)
