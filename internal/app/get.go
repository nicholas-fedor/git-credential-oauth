// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"io"
	"net/http"
	"time"

	"github.com/nicholas-fedor/git-credential-oauth/internal/browser"
	"github.com/nicholas-fedor/git-credential-oauth/internal/forge"
	"github.com/nicholas-fedor/git-credential-oauth/internal/get"
	"github.com/nicholas-fedor/git-credential-oauth/internal/git"
	"github.com/nicholas-fedor/git-credential-oauth/internal/oauth"
)

// newGetter wires the credential get use case.
//
// The built-in forge table is passed unchanged. It names endpoints and scopes
// only, so get completes each client from the operator's git config.
//
// Parameters:
//   - gitPath: absolute path of the git binary.
//   - acquirer: token source.
//   - log: diagnostic logger.
//
// Returns:
//   - get.Service: getter that cannot mutate git config.
func newGetter(
	gitPath string,
	acquirer *oauth.ConfigAcquirer,
	log logger,
) get.Service {
	return get.Service{
		Deps: get.Deps{
			Config: git.New(gitPath),
			Auth:   acquirer,
			Forges: forge.New(),
			Log:    log,
			Now:    time.Now,
		},
	}
}

// newAcquirer wires the OAuth grant implementation.
//
// Nonce is nil so oauth generates one. The HTTP client is the process
// default client.
//
// Parameters:
//   - diag: stream for device instructions and QR codes.
//   - log: diagnostic logger.
//
// Returns:
//   - *oauth.ConfigAcquirer: acquirer used by get.
func newAcquirer(diag io.Writer, log logger) *oauth.ConfigAcquirer {
	return &oauth.ConfigAcquirer{
		Browser: &browser.Opener{
			LookPath: nil,
			Run:      nil,
			GOOS:     "",
		},
		Callback: newCallbackFactory(),
		Prompter: terminalPrompter{diag: diag},
		Doer:     http.DefaultClient,
		Log:      log,
		Nonce:    nil,
	}
}
