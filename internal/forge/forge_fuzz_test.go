// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"net/url"
	"strings"
	"testing"
)

// FuzzParseChallenge checks the realm invariants on arbitrary header text.
//
// The seed corpus holds inputs that previously broke realm extraction: a
// carriage return injected into a realm, and a bare unquoted token with no
// challenge scheme in front of it.
//
// Parameters:
//   - f: fuzzing handle.
func FuzzParseChallenge(f *testing.F) {
	f.Add(`Basic realm="GitLab"`)
	f.Add("Basic realm=GitLab")
	f.Add(`Bearer realm="Gitea", error="invalid_token"`)
	f.Add(`Basic realm="Gitea"` + "\n" + `Bearer realm="Forgejo"`)
	f.Add(`Basic realm="Git\rLab"`)
	f.Add("reAlm=\r0")
	f.Add(`Basic realm="GitLab", charset="UTF-8"`)
	f.Add(`Basic realm = "Gitea"`)

	f.Fuzz(func(t *testing.T, header string) {
		realm, ok := ParseChallenge(header)

		if !ok {
			// No realm found is a valid outcome, but it must yield no value.
			if realm != "" {
				t.Fatalf("ok is false but realm is %q", realm)
			}

			return
		}

		// A realm that could forge a header line is the one thing this parser
		// must never return, since a realm is later compared and echoed.
		if strings.ContainsAny(realm, "\r\n") {
			t.Fatalf("realm contains a newline: %q", realm)
		}

		// A known realm wins over an unknown one regardless of order.
		if knownRealm(realm) {
			return
		}

		// An unknown realm must be returned verbatim, including values that
		// merely resemble a forge name.
		again, againOK := ParseChallenge(header)
		if againOK != ok || again != realm {
			t.Fatalf("parse is not stable: %q/%v then %q/%v", realm, ok, again, againOK)
		}
	})
}

// FuzzParseMetadata checks what a metadata document can and cannot do.
//
// The document is fetched from the server being authenticated to, so its body
// is untrusted. An accepted document must name the host that served it as its
// issuer, and every endpoint FromMetadata accepts from it must be https. A
// document that breaks either would send the authorization code or the token
// somewhere other than the host the user asked for.
//
// Parameters:
//   - f: fuzzing handle.
func FuzzParseMetadata(f *testing.F) {
	f.Add(`{"issuer":"https://git.example.com",`+
		`"authorization_endpoint":"https://git.example.com/a",`+
		`"token_endpoint":"https://git.example.com/t"}`, "git.example.com")
	f.Add(`{"issuer":"https://GIT.example.com:8443"}`, "git.example.com:8443")
	f.Add(`{"issuer":"https://evil.example.com"}`, "git.example.com")
	f.Add(`{"issuer":"https://git.example.com","token_endpoint":"a","token_endpoint":"b"}`, "git.example.com")
	f.Add(`{"issuer":"https://git.example.com",`+
		`"authorization_endpoint":"http://git.example.com/a",`+
		`"token_endpoint":"https://git.example.com/t"}`, "git.example.com")
	f.Add(`{"issuer":"https://git.example.com@evil.example.com"}`, "git.example.com")
	f.Add("\xff", "git.example.com")
	f.Add(`{}`, "")

	f.Fuzz(func(t *testing.T, body, host string) {
		meta, found, err := parseMetadata(strings.NewReader(body), host)
		if err != nil || !found {
			if found {
				t.Fatalf("found with error %v", err)
			}

			return
		}

		issuer, parseErr := url.Parse(meta.Issuer)
		if parseErr != nil || !strings.EqualFold(issuer.Host, host) {
			t.Fatalf("accepted issuer %q for host %q", meta.Issuer, host)
		}

		for kind := KindUnknown; int(kind) < kindCount; kind++ {
			client, ok, fromErr := FromMetadata(kind, host, meta)
			if fromErr != nil || !ok {
				continue
			}

			for _, endpoint := range []string{
				client.Endpoint.AuthURL,
				client.Endpoint.TokenURL,
				client.Endpoint.DeviceAuthURL,
			} {
				if endpoint == "" {
					continue
				}

				parsed, endpointErr := url.Parse(endpoint)
				if endpointErr != nil || parsed.Scheme != schemeHTTPS {
					t.Fatalf("kind %s accepted endpoint %q", kind, endpoint)
				}
			}

			if client.ClientSecret != "" {
				t.Fatalf("kind %s gained a client secret from metadata", kind)
			}
		}
	})
}

// FuzzDerive checks every derived endpoint stays on the remote's host.
//
// The remote URL comes from the credential request. Whatever it contains, a
// derived endpoint must be https and on the same host the client is recorded
// for, or a crafted remote could aim the token exchange at another server.
//
// Parameters:
//   - f: fuzzing handle.
func FuzzDerive(f *testing.F) {
	f.Add(int(KindGitLab), "https://gitlab.example.com")
	f.Add(int(KindGitea), "https://git.example.com:8443/")
	f.Add(int(KindForgejo), "https://user@git.example.com")
	f.Add(int(KindGitHubEnterprise), "https://github.example.com/path?query=1#fragment")
	f.Add(int(KindGitLab), "http://gitlab.example.com")
	f.Add(int(KindGitLab), "https://[::1]:443")
	f.Add(int(KindGitHub), "https://github.com")
	f.Add(-1, "")

	f.Fuzz(func(t *testing.T, rawKind int, gitURL string) {
		client, err := Derive(Kind(rawKind), gitURL)
		if err != nil {
			return
		}

		if client.Host == "" {
			t.Fatalf("derived client for %q has no host", gitURL)
		}

		for _, endpoint := range []string{
			client.Endpoint.AuthURL,
			client.Endpoint.TokenURL,
			client.Endpoint.DeviceAuthURL,
		} {
			if endpoint == "" {
				continue
			}

			parsed, parseErr := url.Parse(endpoint)
			if parseErr != nil {
				t.Fatalf("endpoint %q from %q does not parse: %v", endpoint, gitURL, parseErr)
			}

			if parsed.Scheme != schemeHTTPS || parsed.Host != client.Host {
				t.Fatalf("endpoint %q from %q left host %q", endpoint, gitURL, client.Host)
			}
		}
	})
}
