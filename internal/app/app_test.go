// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/nicholas-fedor/git-credential-oauth/internal/get"
)

func TestCodeFor(t *testing.T) {
	t.Parallel()

	if codeFor(&get.MissingConfigError{Host: "example.test"}) != exitMissingConfig {
		t.Fatal("missing config must exit 2")
	}

	if codeFor(errors.New("boom")) != exitFailure {
		t.Fatal("other errors must exit 1")
	}
}

// TestDebugGateDefaultsToQuiet checks logging is off unless asked for.
//
// A credential helper is invoked by git, so unsolicited stderr output is noise
// in someone else's command.
func TestDebugGateDefaultsToQuiet(t *testing.T) {
	t.Parallel()

	gate := newDebugGate()
	if gate.Enabled() {
		t.Fatal("a new gate must start closed")
	}
}

// TestDebugGateTogglesBothWays checks the gate follows the flag.
func TestDebugGateTogglesBothWays(t *testing.T) {
	t.Parallel()

	gate := newDebugGate()

	gate.Set(true)
	if !gate.Enabled() {
		t.Fatal("gate did not open")
	}

	gate.Set(false)
	if gate.Enabled() {
		t.Fatal("gate did not close")
	}
}

// TestGatedLoggerFollowsTheGate checks one logger serves both modes.
//
// The logger is wired into every service before cobra parses anything, so it
// cannot be rebuilt after the flag is known. Switching the sink per call is
// what lets the same value work either way.
func TestGatedLoggerFollowsTheGate(t *testing.T) {
	t.Parallel()

	var sink bytes.Buffer

	gate := newDebugGate()
	adapter := newGatedLogger(&sink, gate)

	adapter.Debug(t.Context(), "before the flag")
	if sink.Len() != 0 {
		t.Fatalf("debug output before the flag: %q", sink.String())
	}

	gate.Set(true)
	adapter.Debug(t.Context(), "after the flag")

	if !strings.Contains(sink.String(), "after the flag") {
		t.Fatalf("debug output after the flag: %q", sink.String())
	}

	gate.Set(false)
	sink.Reset()

	adapter.Debug(t.Context(), "after the flag was cleared")
	if sink.Len() != 0 {
		t.Fatalf("debug output after clearing: %q", sink.String())
	}
}

// TestNewDebugLoggerIsOpen checks the convenience constructor opts in.
func TestNewDebugLoggerIsOpen(t *testing.T) {
	t.Parallel()

	var sink bytes.Buffer

	newDebugLogger(&sink).Debug(t.Context(), "an always-on line")

	if !strings.Contains(sink.String(), "an always-on line") {
		t.Fatalf("stdout = %q", sink.String())
	}
}

func TestWriteDiagnosticDropsWriteError(t *testing.T) {
	t.Parallel()

	writeDiagnostic(errWriter{}, "ignored\n")
}

func TestRunMissingGit(t *testing.T) {
	t.Setenv("PATH", "")

	if code := Run(t.Context()); code != exitFailure {
		t.Fatalf("Run() = %d, want %d", code, exitFailure)
	}
}

func TestRunCapability(t *testing.T) {
	oldArgs := os.Args
	oldOut := os.Stdout
	t.Cleanup(func() {
		os.Args = oldArgs
		os.Stdout = oldOut
	})

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	os.Stdout = writer
	os.Args = []string{"git-credential-oauth", "capability"}

	code := Run(t.Context())

	_ = writer.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, reader)
	_ = reader.Close()

	if code != exitOK {
		t.Fatalf("Run() = %d, want %d, output %q", code, exitOK, buf.String())
	}

	if buf.String() != "version 0\ncapability authtype\n" {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestRunMissingConfig(t *testing.T) {
	oldArgs := os.Args
	oldIn := os.Stdin
	t.Cleanup(func() {
		os.Args = oldArgs
		os.Stdin = oldIn
	})

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	_, _ = io.WriteString(writer, "protocol=https\nhost=no-such-host.example\n\n")
	_ = writer.Close()

	os.Stdin = reader
	os.Args = []string{"git-credential-oauth", "get"}

	if code := Run(t.Context()); code != exitMissingConfig {
		t.Fatalf("Run() = %d, want %d", code, exitMissingConfig)
	}
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
