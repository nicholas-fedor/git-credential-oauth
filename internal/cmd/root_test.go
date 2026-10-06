// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunHelpWritesTheHelpText checks a bare invocation prints help.
//
// Cobra's Help does not report write failures, so this covers the output
// only.
func TestRunHelpWritesTheHelpText(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	command := &cobra.Command{Use: "x", Long: "help text"}
	command.SetOut(&out)

	require.NoError(t, runHelp(command, nil))
	assert.Contains(t, out.String(), "help text")
}

// TestSilenceKeepsCobraOffTheProtocolStream checks usage and errors are quiet.
//
// Git reads standard output as the credential response, so cobra must not
// print usage or its own error lines there.
func TestSilenceKeepsCobraOffTheProtocolStream(t *testing.T) {
	t.Parallel()

	command := &cobra.Command{}
	silence(command)

	assert.True(t, command.SilenceUsage)
	assert.True(t, command.SilenceErrors)
}

// TestDocRootBuildsTheShippedTree checks the generator sees every command.
func TestDocRootBuildsTheShippedTree(t *testing.T) {
	t.Parallel()

	root := DocRoot(t.Context())

	names := make([]string, 0, len(root.Commands()))
	for _, command := range root.Commands() {
		names = append(names, command.Name())
	}

	assert.ElementsMatch(t,
		[]string{"capability", "configure", "erase", "get", "store", "unconfigure", "version"},
		names)
}
