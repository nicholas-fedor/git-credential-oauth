// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package helper

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/options"
	"github.com/nicholas-fedor/git-credential-oauth/internal/helper"
)

// recordingHelper records the options it was given.
type recordingHelper struct {
	configured   []helper.Options
	unconfigured []helper.Options
}

// Configure records opts.
func (h *recordingHelper) Configure(_ context.Context, opts helper.Options) error {
	h.configured = append(h.configured, opts)

	return nil
}

// Unconfigure records opts.
func (h *recordingHelper) Unconfigure(_ context.Context, opts helper.Options) error {
	h.unconfigured = append(h.unconfigured, opts)

	return nil
}

// brokenWriter fails every write.
type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errBroken }

var errBroken = errors.New("broken pipe")

// TestRunConfigureRefusesAnUnknownStorageBeforeTouchingGit checks validation
// comes first.
func TestRunConfigureRefusesAnUnknownStorageBeforeTouchingGit(t *testing.T) {
	t.Parallel()

	service := &recordingHelper{}
	opts := options.New()
	opts.Storage = "keychain"

	err := runConfigure(t.Context(), service, &bytes.Buffer{}, opts)
	require.Error(t, err)
	assert.Empty(t, service.configured)
}

// TestRunConfigureReportsAFailedMessage covers a closed standard error.
func TestRunConfigureReportsAFailedMessage(t *testing.T) {
	t.Parallel()

	service := &recordingHelper{}

	err := runConfigure(t.Context(), service, brokenWriter{}, options.New())
	require.ErrorIs(t, err, errBroken)
	require.Len(t, service.configured, 1, "the configuration was still written")
}

// TestRunUnconfigureIgnoresTheStorageAndDeviceChoice checks the removal plan
// does not depend on how the helper was configured.
func TestRunUnconfigureIgnoresTheStorageAndDeviceChoice(t *testing.T) {
	t.Parallel()

	service := &recordingHelper{}
	opts := options.New()
	opts.Device = true

	var stderr bytes.Buffer

	require.NoError(t, runUnconfigure(t.Context(), service, &stderr, opts))
	require.Len(t, service.unconfigured, 1)
	assert.Equal(t, helper.StorageAuto, service.unconfigured[0].Storage)
	assert.Equal(t, "unconfigured successfully\n", stderr.String())

	err := runUnconfigure(t.Context(), service, brokenWriter{}, opts)
	require.ErrorIs(t, err, errBroken)
}
