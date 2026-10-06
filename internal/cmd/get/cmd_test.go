// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/options"
	"github.com/nicholas-fedor/git-credential-oauth/internal/credential"
	"github.com/nicholas-fedor/git-credential-oauth/internal/get"
)

// fixedGetter returns a fixed response.
type fixedGetter struct {
	response *credential.Response
}

// Get returns the configured response.
func (g fixedGetter) Get(context.Context, credential.Request, get.Options) (credential.Response, error) {
	return *g.response, nil
}

// TestRunRefusesAnInjectedAttribute checks nothing reaches Git.
//
// A token containing a newline would add attributes of its own to Git's
// input. The command must fail without writing a partial response.
func TestRunRefusesAnInjectedAttribute(t *testing.T) {
	t.Parallel()

	response := credential.NewResponse()
	response.Add(credential.Username, "oauth2")
	response.Add(credential.Password, "token\nusername=attacker")

	var stdout bytes.Buffer

	err := run(t.Context(), strings.NewReader("protocol=https\nhost=x\n\n"), &stdout,
		fixedGetter{response: response}, options.New())
	require.ErrorIs(t, err, credential.ErrUnsafeAttribute)
	assert.Empty(t, stdout.String())
}
