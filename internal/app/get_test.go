// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/git-credential-oauth/internal/browser"
	"github.com/nicholas-fedor/git-credential-oauth/internal/forge"
	"github.com/nicholas-fedor/git-credential-oauth/internal/git"
)

// TestNewGetterWiresEveryDependency checks nothing is left nil.
//
// A nil registry or acquirer is reported as an error at request time, which
// would turn a wiring mistake into a failure on every credential request.
func TestNewGetterWiresEveryDependency(t *testing.T) {
	t.Parallel()

	acquirer := newAcquirer(&bytes.Buffer{}, newLogger())
	getter := newGetter("/usr/bin/git", acquirer, newLogger())

	assert.IsType(t, &git.Git{}, getter.Deps.Config)
	assert.IsType(t, &forge.Table{}, getter.Deps.Forges)
	assert.Same(t, acquirer, getter.Deps.Auth)
	assert.NotNil(t, getter.Deps.Log)
	assert.NotNil(t, getter.Deps.Now)
}

// TestNewAcquirerWiresTheGrantDependencies checks the browser grant can run.
func TestNewAcquirerWiresTheGrantDependencies(t *testing.T) {
	t.Parallel()

	diag := &bytes.Buffer{}
	acquirer := newAcquirer(diag, newLogger())

	assert.IsType(t, &browser.Opener{}, acquirer.Browser)
	assert.NotNil(t, acquirer.Callback)
	assert.Equal(t, terminalPrompter{diag: diag}, acquirer.Prompter)
	assert.NotNil(t, acquirer.Doer)
	assert.Nil(t, acquirer.Nonce, "the default random state is used")
}
