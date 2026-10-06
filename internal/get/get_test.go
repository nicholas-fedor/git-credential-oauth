// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/forge"
	"github.com/nicholas-fedor/git-credential-oauth/internal/oauth"
)

// fakeConfig answers git config lookups from a map.
type fakeConfig struct {
	err    error
	values map[string]string
	urls   []string
}

// Get returns the value stored for key and records the URL it was asked for.
func (c *fakeConfig) Get(_ context.Context, key, rawURL string) (string, bool, error) {
	c.urls = append(c.urls, rawURL)

	if c.err != nil {
		return "", false, c.err
	}

	value, found := c.values[key]

	return value, found, nil
}

// recordingLogger keeps every warning it receives.
type recordingLogger struct {
	warnings []string
	mu       sync.Mutex
}

func (l *recordingLogger) Debug(context.Context, string, ...any) {}
func (l *recordingLogger) Info(context.Context, string, ...any)  {}
func (l *recordingLogger) Error(context.Context, string, ...any) {}

// Warn records the message and its key-value pairs.
func (l *recordingLogger) Warn(_ context.Context, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.warnings = append(l.warnings, fmt.Sprint(append([]any{msg}, args...)...))
}

var errLookup = errors.New("lookup failed")

// TestReadOverridesLoadsEveryKey checks each key reaches its field.
//
// A key that is read but not applied would be silently ignored, which for the
// client secret or an endpoint means a request that fails for no visible
// reason.
func TestReadOverridesLoadsEveryKey(t *testing.T) {
	t.Parallel()

	config := &fakeConfig{values: map[string]string{
		keyClientID:      "  id  ",
		keyClientSecret:  "secret",
		keyScopes:        "a b",
		keyAuthURL:       "/authorize",
		keyTokenURL:      "/token",
		keyDeviceAuthURL: "/device",
		keyRedirectURL:   "http://127.0.0.1:7171/cb",
		keyExpiryMargin:  "5m",
	}}
	svc := Service{Deps: Deps{Config: config}}

	got, err := svc.readOverrides(t.Context(), "https://git.example.com")
	require.NoError(t, err)

	assert.Equal(t, configOverrides{
		ClientID:           "id",
		ClientSecret:       "secret",
		Scopes:             "a b",
		AuthURL:            "/authorize",
		TokenURL:           "/token",
		DeviceAuthURL:      "/device",
		RedirectURL:        "http://127.0.0.1:7171/cb",
		ExpiryMargin:       "5m",
		ClientIDFound:      true,
		ClientSecretFound:  true,
		ScopesFound:        true,
		AuthURLFound:       true,
		TokenURLFound:      true,
		DeviceAuthURLFound: true,
		RedirectURLFound:   true,
		ExpiryMarginFound:  true,
	}, got)

	for _, rawURL := range config.urls {
		assert.Equal(t, "https://git.example.com", rawURL, "every key is matched against the remote")
	}
}

// TestReadOverridesLeavesUnsetKeysUnfound checks absence is not a value.
func TestReadOverridesLeavesUnsetKeysUnfound(t *testing.T) {
	t.Parallel()

	svc := Service{Deps: Deps{Config: &fakeConfig{values: map[string]string{}}}}

	got, err := svc.readOverrides(t.Context(), "https://git.example.com")
	require.NoError(t, err)
	assert.Equal(t, configOverrides{}, got)
}

// TestReadOverridesStopsOnALookupFailure checks an error is not treated as unset.
//
// Reading a broken config as "nothing configured" would fall back to defaults
// the operator meant to replace.
func TestReadOverridesStopsOnALookupFailure(t *testing.T) {
	t.Parallel()

	svc := Service{Deps: Deps{Config: &fakeConfig{err: errLookup}}}

	_, err := svc.readOverrides(t.Context(), "https://git.example.com")
	require.ErrorIs(t, err, errLookup)
	assert.Contains(t, err.Error(), keyClientID)
}

// TestReadKeyWithoutAConfigReader treats every key as unset.
func TestReadKeyWithoutAConfigReader(t *testing.T) {
	t.Parallel()

	value, found, err := Service{}.readKey(t.Context(), keyClientID, "https://x.example")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, value)
}

// TestAcquireWithoutAnAcquirer fails rather than panics.
func TestAcquireWithoutAnAcquirer(t *testing.T) {
	t.Parallel()

	_, err := Service{}.acquire(t.Context(), &resolution{})
	require.ErrorIs(t, err, errNilAcquirer)
}

// TestWrapAcquireKeepsTheDeviceSentinel covers both wrapping paths.
func TestWrapAcquireKeepsTheDeviceSentinel(t *testing.T) {
	t.Parallel()

	device := wrapAcquire(fmt.Errorf("inner: %w", oauth.ErrDeviceUnsupported))
	require.ErrorIs(t, device, oauth.ErrDeviceUnsupported)
	assert.Contains(t, device.Error(), "device flow unsupported")

	other := wrapAcquire(errLookup)
	require.ErrorIs(t, other, errLookup)
	assert.Contains(t, other.Error(), "acquire oauth token")
}

// TestNoteWarningsReportsBothFindings covers the two non-fatal warnings.
func TestNoteWarningsReportsBothFindings(t *testing.T) {
	t.Parallel()

	log := &recordingLogger{}
	svc := Service{Deps: Deps{Log: log}}

	svc.noteWarnings(t.Context(), &resolution{
		Detected:      forge.Detected{Kind: forge.KindForgejo},
		Scopes:        []string{"read_repository", "read:repository"},
		MarginRaw:     "soon",
		MarginInvalid: true,
	})

	require.Len(t, log.warnings, 2)
	assert.Contains(t, log.warnings[0], "invalid oauth expiry margin")
	assert.Contains(t, log.warnings[0], "soon")
	assert.Contains(t, log.warnings[1], "read_repository")
	assert.NotContains(t, log.warnings[1], "read:repository")
}

// TestNoteWarningsIsQuietWhenNothingIsWrong checks no false alarms.
func TestNoteWarningsIsQuietWhenNothingIsWrong(t *testing.T) {
	t.Parallel()

	log := &recordingLogger{}
	svc := Service{Deps: Deps{Log: log}}

	svc.noteWarnings(t.Context(), &resolution{
		Detected: forge.Detected{Kind: forge.KindGitLab},
		Scopes:   []string{"read_repository"},
	})

	assert.Empty(t, log.warnings, "GitLab scopes are right for GitLab")
}

// TestWarningsTolerateANilLogger checks a missing logger is not a crash.
func TestWarningsTolerateANilLogger(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		warnInvalidMargin(t.Context(), nil, "x")
		noteScopeShape(t.Context(), nil, forge.KindGitea, []string{"read_repository"})
	})
}

// TestNowUsesTheInjectedClock covers both clock sources.
func TestNowUsesTheInjectedClock(t *testing.T) {
	t.Parallel()

	fixed := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	assert.Equal(t, fixed, Service{Deps: Deps{Now: func() time.Time { return fixed }}}.now())

	before := time.Now()
	assert.False(t, Service{}.now().Before(before), "the default clock is the wall clock")
}
