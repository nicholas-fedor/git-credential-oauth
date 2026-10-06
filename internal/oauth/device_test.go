// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// countingPrompter counts prompts and returns a fixed error.
type countingPrompter struct {
	err   error
	calls int
}

// PromptDevice counts the call.
func (p *countingPrompter) PromptDevice(context.Context, string, string, string) error {
	p.calls++

	return p.err
}

// TestAcquireDeviceRefusesAnUnsupportedForge checks nothing is sent.
//
// A forge without the device grant has no endpoint to call, so the request
// must fail before any network traffic and before the user is prompted.
func TestAcquireDeviceRefusesAnUnsupportedForge(t *testing.T) {
	t.Parallel()

	prompter := &countingPrompter{}
	acq := &ConfigAcquirer{Prompter: prompter}

	_, err := acq.acquireDevice(t.Context(), oauth2.Config{}, Input{Device: true, DeviceFlow: false})
	require.ErrorIs(t, err, ErrDeviceUnsupported)
	assert.Zero(t, prompter.calls)
}

// TestAcquireDeviceNeedsAPrompter fails rather than panics.
func TestAcquireDeviceNeedsAPrompter(t *testing.T) {
	t.Parallel()

	_, err := (&ConfigAcquirer{}).acquireDevice(t.Context(), oauth2.Config{}, Input{DeviceFlow: true})
	require.ErrorIs(t, err, errNilPrompter)
}

// TestAcquireDeviceReportsADeviceAuthFailure covers a forge that refuses.
func TestAcquireDeviceReportsADeviceAuthFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"device_flow_disabled"}`))
	}))
	t.Cleanup(server.Close)

	prompter := &countingPrompter{}
	acq := &ConfigAcquirer{Prompter: prompter, Doer: server.Client()}
	cfg := oauth2.Config{ClientID: "id", Endpoint: oauth2.Endpoint{DeviceAuthURL: server.URL}}

	_, err := acq.acquireDevice(t.Context(), cfg, Input{DeviceFlow: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device auth")
	assert.Zero(t, prompter.calls, "the user is not prompted for a code that was never issued")
}

// TestAcquireDeviceStopsWhenThePromptFails covers a terminal that cannot be
// written to.
func TestAcquireDeviceStopsWhenThePromptFails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"device_code":"d","user_code":"U","verification_uri":"https://x/device",` +
			`"expires_in":900,"interval":1}`))
	}))
	t.Cleanup(server.Close)

	prompter := &countingPrompter{err: errStub}
	acq := &ConfigAcquirer{Prompter: prompter, Doer: server.Client()}
	cfg := oauth2.Config{ClientID: "id", Endpoint: oauth2.Endpoint{DeviceAuthURL: server.URL}}

	_, err := acq.acquireDevice(t.Context(), cfg, Input{DeviceFlow: true})
	require.ErrorIs(t, err, errStub)
	assert.Equal(t, 1, prompter.calls)
}
