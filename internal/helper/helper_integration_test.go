// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package helper_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/git"
	"github.com/nicholas-fedor/git-credential-oauth/internal/helper"
	mockHelper "github.com/nicholas-fedor/git-credential-oauth/internal/helper/mocks"
)

func TestUnconfigureIgnoresExitNotFound(t *testing.T) {
	t.Parallel()

	runner, calls := scriptedRunner(t, func(int) error {
		return notFoundErr()
	})
	log, counts := countingLog(t)
	svc := &helper.Service{
		Git: runner,
		Log: log,
	}
	opts := helper.Options{
		Storage: helper.StorageLibsecret,
		Device:  true,
		GOOS:    "linux",
	}

	err := svc.Unconfigure(t.Context(), opts)
	require.NoError(t, err)
	assert.Equal(t, helper.PlanUnconfigure(), *calls)

	assert.Positive(t, counts.infos)
	assert.Equal(t, len(*calls), counts.warns)
	assert.Zero(t, counts.errs)
}

func TestUnconfigureIgnoresWrappedExitNotFound(t *testing.T) {
	t.Parallel()

	runner, calls := scriptedRunner(t, func(int) error {
		return fmt.Errorf("runner: %w", notFoundErr())
	})
	svc := &helper.Service{
		Git: runner,
		Log: nil,
	}

	err := svc.Unconfigure(t.Context(), helper.Options{
		Storage: helper.StorageNone,
		Device:  false,
		GOOS:    "darwin",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, *calls)
}

func TestUnconfigurePropagatesNeverStarted(t *testing.T) {
	t.Parallel()

	neverStarted := &git.Error{
		Args:     []string{"config"},
		ExitCode: git.ExitNeverStarted,
		Stderr:   "git not started",
		Err:      errExecutableNotFound,
	}
	runner, calls := scriptedRunner(t, func(call int) error {
		if call == 1 {
			return neverStarted
		}

		return nil
	})
	svc := &helper.Service{
		Git: runner,
		Log: nil,
	}

	err := svc.Unconfigure(t.Context(), helper.Options{
		Storage: helper.StorageAuto,
		Device:  false,
		GOOS:    "windows",
	})
	require.Error(t, err)

	var gitErr *git.Error

	require.ErrorAs(t, err, &gitErr)
	assert.Equal(t, git.ExitNeverStarted, gitErr.ExitCode)
	assert.Len(t, *calls, 1)
}

func TestUnconfigurePropagatesGenericError(t *testing.T) {
	t.Parallel()

	runner, calls := scriptedRunner(t, func(call int) error {
		if call == 2 {
			return errGenericExit
		}

		return nil
	})
	svc := &helper.Service{
		Git: runner,
		Log: nil,
	}

	err := svc.Unconfigure(t.Context(), helper.Options{
		Storage: helper.StorageStore,
		Device:  true,
		GOOS:    "linux",
	})
	require.ErrorIs(t, err, errGenericExit)
	assert.Len(t, *calls, 2)
}

func TestConfigureIgnoresExitNotFound(t *testing.T) {
	t.Parallel()

	runner, calls := scriptedRunner(t, func(call int) error {
		if call == 1 {
			return notFoundErr()
		}

		return nil
	})
	svc := &helper.Service{
		Git: runner,
		Log: nil,
	}
	opts := helper.Options{
		Storage: helper.StorageCache,
		Device:  false,
		GOOS:    "linux",
	}

	err := svc.Configure(t.Context(), opts)
	require.NoError(t, err)

	want, planErr := helper.PlanConfigure(opts)
	require.NoError(t, planErr)
	assert.Equal(t, want, *calls)
}

func TestConfigureNilGit(t *testing.T) {
	t.Parallel()

	svc := &helper.Service{
		Git: nil,
		Log: nil,
	}

	err := svc.Configure(t.Context(), helper.Options{
		Storage: helper.StorageAuto,
		Device:  false,
		GOOS:    "linux",
	})
	require.ErrorIs(t, err, helper.ErrNilGit)
}

func TestNilService(t *testing.T) {
	t.Parallel()

	var svc *helper.Service

	err := svc.Unconfigure(t.Context(), helper.Options{
		Storage: helper.StorageNone,
		Device:  false,
		GOOS:    "",
	})
	require.ErrorIs(t, err, helper.ErrNilGit)
}

func TestConfigureNilLog(t *testing.T) {
	t.Parallel()

	runner, calls := scriptedRunner(t, nil)
	svc := &helper.Service{
		Git: runner,
		Log: nil,
	}
	opts := helper.Options{
		Storage: helper.StorageNone,
		Device:  true,
		GOOS:    "linux",
	}

	err := svc.Configure(t.Context(), opts)
	require.NoError(t, err)

	want, planErr := helper.PlanConfigure(opts)
	require.NoError(t, planErr)
	assert.Equal(t, want, *calls)
}

func TestConfigureRejectsUnknownStorage(t *testing.T) {
	t.Parallel()

	runner := mockHelper.NewMockGitRunner(t)
	svc := &helper.Service{
		Git: runner,
		Log: nil,
	}

	err := svc.Configure(t.Context(), helper.Options{
		Storage: helper.Storage("nope"),
		Device:  false,
		GOOS:    "linux",
	})
	require.ErrorIs(t, err, helper.ErrUnknownStorage)
	runner.AssertNotCalled(t, "Run", mock.Anything, mock.Anything)
}

type logCounts struct {
	infos int
	warns int
	errs  int
}

func countingLog(t *testing.T) (*mockHelper.MockLogger, *logCounts) {
	t.Helper()

	log := mockHelper.NewMockLogger(t)
	counts := &logCounts{}
	log.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything).Run(
		func(context.Context, string, ...any) {
			counts.infos++
		},
	).Return()
	log.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything).Run(
		func(context.Context, string, ...any) {
			counts.warns++
		},
	).Return()
	log.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything).Run(
		func(context.Context, string, ...any) {
			counts.errs++
		},
	).Return().Maybe()

	return log, counts
}

func scriptedRunner(
	t *testing.T,
	fail func(call int) error,
) (*mockHelper.MockGitRunner, *[][]string) {
	t.Helper()

	runner := mockHelper.NewMockGitRunner(t)
	calls := [][]string{}
	runner.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, args ...string) error {
			calls = append(calls, append([]string(nil), args...))
			if fail == nil {
				return nil
			}

			return fail(len(calls))
		},
	)

	return runner, &calls
}

func notFoundErr() error {
	return &git.Error{
		Args:     []string{"config", "--global", "--unset-all", "credential.helper"},
		ExitCode: git.ExitNotFound,
		Stderr:   "",
		Err:      errExitStatus,
	}
}

var (
	errExecutableNotFound = errors.New("executable file not found")
	errGenericExit        = errors.New("exit status 5")
	errExitStatus         = errors.New("exit status 5")
)
