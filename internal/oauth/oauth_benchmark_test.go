// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth_test

import (
	"testing"

	"golang.org/x/oauth2"

	"github.com/nicholas-fedor/git-credential-oauth/internal/oauth"
)

func BenchmarkAuthCodeRequestURL(b *testing.B) {
	cfg := oauth2.Config{
		ClientID:    "client-id",
		RedirectURL: "http://127.0.0.1:8080/callback",
		Scopes:      []string{"repo", "read:user"},
		Endpoint: oauth2.Endpoint{
			AuthURL: "https://example.com/login/oauth/authorize",
		},
	}

	const (
		state    = "state-value"
		verifier = "verifier-value-verifier-value-verifier"
		suffix   = "&login=octocat"
	)

	var sink string

	b.ReportAllocs()

	for b.Loop() {
		sink = oauth.AuthCodeRequestURL(cfg, state, verifier, suffix)
	}

	if sink == "" {
		b.Fatal("empty authorization url")
	}
}
