// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package helper

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/git"
)

// recordingGit records every git invocation and fails on chosen ones.
type recordingGit struct {
	fail  map[int]error
	calls [][]string
}

// Run records args and returns the error configured for this call.
func (g *recordingGit) Run(_ context.Context, args ...string) error {
	g.calls = append(g.calls, args)

	return g.fail[len(g.calls)-1]
}

// recordingLog keeps message names by level.
type recordingLog struct {
	info []string
	warn []string
}

func (l *recordingLog) Info(_ context.Context, msg string, _ ...any) { l.info = append(l.info, msg) }

func (l *recordingLog) Warn(_ context.Context, msg string, _ ...any) { l.warn = append(l.warn, msg) }
func (l *recordingLog) Error(context.Context, string, ...any)        {}

var errGitFailed = errors.New("git failed")

// TestConfigureRunsThePlanInOrder checks configure executes what it planned.
func TestConfigureRunsThePlanInOrder(t *testing.T) {
	t.Parallel()

	runner := &recordingGit{}
	svc := &Service{Git: runner}
	opts := Options{Storage: StorageLibsecret, GOOS: "linux"}

	require.NoError(t, svc.Configure(t.Context(), opts))

	plan, err := PlanConfigure(opts)
	require.NoError(t, err)
	assert.Equal(t, plan, runner.calls)
}

// TestRunPlanStopsAtTheFirstRealFailure checks later steps are skipped.
//
// Configuring after a failed unset would append to a list that was meant to be
// replaced, so a failure ends the plan.
func TestRunPlanStopsAtTheFirstRealFailure(t *testing.T) {
	t.Parallel()

	runner := &recordingGit{fail: map[int]error{1: errGitFailed}}
	svc := &Service{Git: runner}

	err := runPlan(t.Context(), svc, [][]string{{"a"}, {"b"}, {"c"}})
	require.ErrorIs(t, err, errGitFailed)
	assert.Len(t, runner.calls, 2)
}

// TestRunArgsToleratesAnAbsentKey checks exit 5 from an unset is not a failure.
func TestRunArgsToleratesAnAbsentKey(t *testing.T) {
	t.Parallel()

	log := &recordingLog{}
	notFound := fmt.Errorf("wrapped: %w", &git.Error{ExitCode: git.ExitNotFound})
	svc := &Service{Git: &recordingGit{fail: map[int]error{0: notFound}}, Log: log}

	require.NoError(t, runArgs(t.Context(), svc, []string{"config", "--unset-all", "k"}))
	assert.Equal(t, []string{"Running git"}, log.info)
	assert.Equal(t, []string{"Credential helper key was absent"}, log.warn)
}

// TestRunPlanNeedsAGitRunner fails rather than panics.
func TestRunPlanNeedsAGitRunner(t *testing.T) {
	t.Parallel()

	require.ErrorIs(t, runPlan(t.Context(), nil, nil), ErrNilGit)
	require.ErrorIs(t, runPlan(t.Context(), &Service{}, nil), ErrNilGit)
}

// TestLoggingToleratesAMissingLogger covers nil services and loggers.
func TestLoggingToleratesAMissingLogger(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		logInfo(t.Context(), nil, "m")
		logWarn(t.Context(), nil, "m")
		logInfo(t.Context(), &Service{}, "m")
		logWarn(t.Context(), &Service{}, "m")
	})
}
