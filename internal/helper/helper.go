// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package helper

import (
	"context"
	"fmt"
)

// Options controls how the credential helper is configured.
type Options struct {
	Storage Storage
	GOOS    string
	Device  bool
}

// Service applies credential-helper plans through GitRunner.
type Service struct {
	// Git runs each planned argument slice.
	Git GitRunner

	// Log receives operational messages.
	// Nil is a no-op and does not panic.
	Log Logger
}

// Configure installs git-credential-oauth as a global Git credential helper.
//
// Exit code 5 from git is ignored so a missing key does not fail setup.
// A nil Git runner returns [ErrNilGit]. A nil Log is ignored.
//
// Parameters:
//   - ctx: cancellation context passed to each git invocation.
//   - opts: storage selection, device flow, and GOOS for auto.
//
// Returns:
//   - error: non-nil when storage is unknown, Git is nil, or git fails.
func (svc *Service) Configure(ctx context.Context, opts Options) error {
	logInfo(ctx, svc, "Configuring credential helper",
		keyStorage, string(opts.Storage),
		keyDevice, opts.Device,
	)

	plan, err := PlanConfigure(opts)
	if err != nil {
		return err
	}

	return runPlan(ctx, svc, plan)
}

// Unconfigure removes git-credential-oauth and the storage helpers it may have
// set.
//
// The plan does not depend on opts, so a later storage choice cannot leave
// an earlier helper behind. opts is logged. Exit code 5 is ignored.
// Exit codes other than 5, including a process that never started, are
// returned.
//
// Parameters:
//   - ctx: cancellation context passed to each git invocation.
//   - opts: logged for diagnostics. It does not change the plan.
//
// Returns:
//   - error: non-nil when Git is nil or git fails for a reason other than
//     a missing key.
func (svc *Service) Unconfigure(ctx context.Context, opts Options) error {
	logInfo(ctx, svc, "Unconfiguring credential helper",
		keyStorage, string(opts.Storage),
		keyDevice, opts.Device,
	)

	return runPlan(ctx, svc, PlanUnconfigure())
}

// runPlan executes each argument slice in order.
//
// The first error other than a missing key stops the plan.
//
// Parameters:
//   - ctx: cancellation context passed to each git invocation.
//   - svc: service whose Git runner executes the plan.
//   - plan: argument slices, excluding the git binary.
//
// Returns:
//   - error: non-nil when Git is nil or a command fails.
func runPlan(ctx context.Context, svc *Service, plan [][]string) error {
	if svc == nil || svc.Git == nil {
		return ErrNilGit
	}

	for _, args := range plan {
		err := runArgs(ctx, svc, args)
		if err != nil {
			return err
		}
	}

	return nil
}

// runArgs executes one git argument slice.
//
// Exit code 5 is ignored. Exit code -1 and every other error are wrapped
// and returned.
//
// Parameters:
//   - ctx: cancellation context for the invocation.
//   - svc: service whose Git runner and logger are used.
//   - args: git arguments, excluding the git binary.
//
// Returns:
//   - error: non-nil when git fails for a reason other than a missing key.
func runArgs(ctx context.Context, svc *Service, args []string) error {
	logInfo(ctx, svc, "Running git", keyArgs, args)

	err := svc.Git.Run(ctx, args...)
	if err == nil {
		return nil
	}

	if isNotFound(err) {
		logWarn(ctx, svc, "Credential helper key was absent", keyArgs, args)

		return nil
	}

	return fmt.Errorf("run git config: %w", err)
}
