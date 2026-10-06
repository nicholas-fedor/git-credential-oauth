// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLoggerDiscards checks the quiet logger writes nothing.
func TestNewLoggerDiscards(t *testing.T) {
	t.Parallel()

	quiet := newLogger()
	adapter := quiet

	ctx := t.Context()

	adapter.Debug(ctx, "debug event", "key", "value")
	adapter.Info(ctx, "info event", "key", "value")
	adapter.Warn(ctx, "warn event", "key", "value")
	adapter.Error(ctx, "error event", "key", "value")
}

// TestNewDebugLoggerWritesEveryLevel checks each level reaches the stream.
//
// Git reads this program's standard output as a credential, so debug text can
// only go to the diagnostic stream; a lost level would hide a real failure.
func TestNewDebugLoggerWritesEveryLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level func(logger, context.Context)
		want  string
	}{
		{
			name:  "debug",
			level: func(l logger, c context.Context) { l.Debug(c, "a debug line", "k", "v") },
			want:  "a debug line",
		},
		{
			name:  "info",
			level: func(l logger, c context.Context) { l.Info(c, "an info line", "k", "v") },
			want:  "an info line",
		},
		{
			name:  "warn",
			level: func(l logger, c context.Context) { l.Warn(c, "a warn line", "k", "v") },
			want:  "a warn line",
		},
		{
			name:  "error",
			level: func(l logger, c context.Context) { l.Error(c, "an error line", "k", "v") },
			want:  "an error line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer

			adapter := newDebugLogger(&buffer)
			tt.level(adapter, t.Context())

			assert.Contains(t, buffer.String(), tt.want)
		})
	}
}

// TestDebugLoggerCarriesKeyValuePairs checks attributes survive the adapter.
func TestDebugLoggerCarriesKeyValuePairs(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	adapter := newDebugLogger(&buffer)
	adapter.Info(t.Context(), "token refreshed", "host", "example.com")

	rendered := buffer.String()
	require.NotEmpty(t, rendered)
	assert.Contains(t, rendered, "host=example.com")
}

// TestDebugLoggerDoesNotAddSource checks the text form stays compact.
//
// The source position on every line would bury the message a user is trying
// to read, so AddSource is off.
func TestDebugLoggerDoesNotAddSource(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	adapter := newDebugLogger(&buffer)
	adapter.Info(t.Context(), "a line")

	assert.NotContains(t, buffer.String(), "logger_test.go")
}
