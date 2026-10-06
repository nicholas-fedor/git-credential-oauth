// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import "strings"

const (
	// prefixGitLab matches a self-hosted GitLab hostname.
	prefixGitLab = "gitlab."
	// prefixGitea matches a self-hosted Gitea hostname.
	prefixGitea = "gitea."
	// prefixForgejo matches a self-hosted Forgejo hostname.
	prefixForgejo = "forgejo."
	// prefixGitHub matches a GitHub Enterprise hostname.
	prefixGitHub = "github."
	// suffixGoogleSource matches a Google Source hostname.
	suffixGoogleSource = ".googlesource.com"
)

// Detect classifies a Git host.
//
// Order is registry Lookup, then subdomain prefix, then the realm from
// ParseChallenge, then the googlesource.com suffix. github.com and gitea.com
// match the github. and gitea. prefixes, so a registry hit is required to
// keep them off the self-hosted path. An unknown github.example.com host is
// GitHub Enterprise.
//
// Only the forge side of a remote is recorded here. The protocol and host the
// caller already has, and the scheme follows from the protocol, so neither is
// carried on the result.
//
// Parameters:
//   - reg: host registry. A nil registry is treated as a miss.
//   - host: Git hostname, compared as given.
//   - wwwAuth: WWW-Authenticate header, possibly with several challenges.
//
// Returns:
//   - detected: kind, host, and realm.
func Detect(reg Registry, host, wwwAuth string) Detected {
	realm, _ := ParseChallenge(wwwAuth)

	return Detected{
		Kind:  detectKind(reg, host, realm),
		Host:  host,
		Realm: realm,
	}
}

// detectKind applies the detection order and returns the forge kind.
//
// Registry hits win over prefix and realm matches so public apex hosts are
// not classified as self-hosted.
//
// Parameters:
//   - reg: host registry. Nil is a miss.
//   - host: Git hostname, compared as given.
//   - realm: realm already parsed from the WWW-Authenticate header.
//
// Returns:
//   - Kind: detected forge family, or KindUnknown.
func detectKind(reg Registry, host, realm string) Kind {
	// Registry wins so public apex hosts are not prefix hits.
	if kind, found := registryKind(reg, host); found {
		return kind
	}

	if kind, found := prefixKind(host); found {
		return kind
	}

	if kind, found := realmKind(realm); found {
		return kind
	}

	if strings.HasSuffix(host, suffixGoogleSource) {
		return KindGoogleSource
	}

	return KindUnknown
}

// registryKind returns the kind stored for host.
//
// Parameters:
//   - reg: host registry. Nil is a miss.
//   - host: Git hostname to look up.
//
// Returns:
//   - Kind: registry kind when found.
//   - bool: true when the host is in the registry.
func registryKind(reg Registry, host string) (Kind, bool) {
	if reg == nil {
		return KindUnknown, false
	}

	client, ok := reg.Lookup(host)
	if !ok {
		return KindUnknown, false
	}

	return client.Kind, true
}

// prefixKind maps a subdomain prefix to a kind.
//
// The github. prefix is GitHub Enterprise, not public GitHub. gitea.com is
// also a prefix match; callers must consult the registry first.
//
// Parameters:
//   - host: Git hostname, compared as given.
//
// Returns:
//   - Kind: prefix kind when one matches.
//   - bool: true when a prefix matches.
func prefixKind(host string) (Kind, bool) {
	switch {
	case strings.HasPrefix(host, prefixGitLab):
		return KindGitLab, true
	case strings.HasPrefix(host, prefixGitea):
		return KindGitea, true
	case strings.HasPrefix(host, prefixForgejo):
		return KindForgejo, true
	case strings.HasPrefix(host, prefixGitHub):
		return KindGitHubEnterprise, true
	default:
		return KindUnknown, false
	}
}

// realmKind maps an exact realm value to a kind.
//
// realm="NotGitLab" is not GitLab. A GitHub realm on an unknown host is
// GitHub Enterprise, matching the self-hosted GitHub URL layout.
//
// Parameters:
//   - realm: realm parameter from the WWW-Authenticate header.
//
// Returns:
//   - Kind: realm kind when the value is exact.
//   - bool: true when the realm is a known forge realm.
func realmKind(realm string) (Kind, bool) {
	switch realm {
	case realmGitLab:
		return KindGitLab, true
	case realmGitea:
		return KindGitea, true
	case realmForgejo:
		return KindForgejo, true
	case realmGitHub:
		return KindGitHubEnterprise, true
	default:
		return KindUnknown, false
	}
}
