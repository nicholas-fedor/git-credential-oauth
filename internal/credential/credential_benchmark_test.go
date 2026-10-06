// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package credential_test

import (
	"testing"

	"github.com/nicholas-fedor/git-credential-oauth/internal/credential"
)

// BenchmarkParseString measures parsing a representative credential request.
func BenchmarkParseString(b *testing.B) {
	const input = "capability[]=authtype\n" +
		"capability[]=state\n" +
		"protocol=https\n" +
		"host=github.com\n" +
		"path=/nicholas-fedor/git-credential-oauth.git\n" +
		"username=oauth2\n" +
		"wwwauth[]=Basic realm=\"GitHub\"\n" +
		"wwwauth[]=Bearer realm=\"GitHub\"\n" +
		"oauth_refresh_token=refresh-token\n"

	var host string

	for b.Loop() {
		request, err := credential.ParseString(input)
		if err != nil {
			b.Fatal(err)
		}

		host = request.Host
	}

	if host == "" {
		b.Fatal("missing host")
	}
}
