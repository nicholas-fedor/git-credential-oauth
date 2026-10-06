// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import "strconv"

// Kind identifies a Git hosting forge family.
//
// The zero value is [KindUnknown], so an unclassified host is never mistaken
// for a recognized family.
type Kind int

// setupHint names where an operator registers an OAuth application for a
// family.
type setupHint uint8

// kindProfile holds what distinguishes one forge family from another.
//
// A false field is the default, so a family that behaves like the others needs
// no entry at all.
type kindProfile struct {
	// loginHint is the authorization-URL query key for a username hint.
	loginHint string

	// setup selects the registration guidance for the family.
	setup setupHint

	// bearer marks families whose tokens can be sent as a Bearer credential.
	bearer bool

	// publicClientID marks families whose servers register an application for
	// this helper themselves, so an instance works before an operator registers
	// anything.
	publicClientID bool

	// secret marks families whose registered application comes with a client
	// secret that the token endpoint expects on the browser grant.
	secret bool

	// giteaLayout marks families that use the Gitea endpoint paths and the
	// colon-separated scope names rather than GitLab's underscore form.
	giteaLayout bool

	// selfHosted marks families that can be deployed off the vendor cloud.
	selfHosted bool

	// device marks families that support the device authorization grant.
	device bool
}

const (
	// KindUnknown is an unrecognized host.
	KindUnknown Kind = iota

	// KindGitHub is the public github.com host.
	KindGitHub

	// KindGitHubEnterprise is a self-hosted GitHub Enterprise Server.
	KindGitHubEnterprise

	// KindGitLab is a self-hosted or public GitLab instance.
	KindGitLab

	// KindGitea is a Gitea host.
	KindGitea

	// KindForgejo is a Forgejo host.
	KindForgejo

	// KindBitbucket is the public bitbucket.org host.
	KindBitbucket

	// KindGoogleSource is a googlesource.com host.
	KindGoogleSource
)

// kindCount is one past the last declared kind.
//
// It is a plain int because it bounds a sum and an array length, and is never
// itself a valid kind.
const kindCount = int(KindGoogleSource) + 1

const (
	// setupRegister tells the operator to register an application, without
	// naming a page, because the family has no known registration form.
	setupRegister setupHint = iota

	// setupGitHub points at the github.com application form.
	setupGitHub

	// setupGitHubEnterprise points at the application form on the instance.
	setupGitHubEnterprise

	// setupGitLab points at the GitLab application form on the instance.
	setupGitLab

	// setupBitbucket points at the Bitbucket workspace's OAuth consumers.
	setupBitbucket

	// setupGoogle points at the Google Cloud console clients page.
	setupGoogle
)

// kindProfiles holds every fact that differs between forge families.
//
// Keeping these together is what makes adding a family a one-line change.
// Scattering them across per-behavior switches meant that each new kind had to
// be added in seven places, and a switch whose default arm repeated its own
// tail would lint clean while routing the new kind to the wrong behavior.
//
// Index is the Kind value, so a missing entry fails the table test rather than
// silently defaulting.
//
//nolint:exhaustruct_v5 // Every family overrides only the facts that differ.
var kindProfiles = [kindCount]kindProfile{
	KindUnknown: {},
	KindGitHub: {
		loginHint: loginParamGitHub,
		setup:     setupGitHub,
		secret:    true,
	},
	KindGitHubEnterprise: {
		loginHint:  loginParamGitHub,
		setup:      setupGitHubEnterprise,
		secret:     true,
		selfHosted: true,
		device:     true,
	},
	KindGitLab: {
		setup:      setupGitLab,
		selfHosted: true,
		device:     true,
	},
	KindGitea: {
		bearer:         true,
		publicClientID: true,
		giteaLayout:    true,
		selfHosted:     true,
	},
	KindForgejo: {
		bearer:         true,
		publicClientID: true,
		giteaLayout:    true,
		selfHosted:     true,
	},
	KindBitbucket: {
		setup:  setupBitbucket,
		secret: true,
	},
	KindGoogleSource: {
		loginHint: loginParamGoogle,
		setup:     setupGoogle,
		bearer:    true,
		secret:    true,
	},
}

// kindNames holds the identifier name of every declared kind, indexed by value.
//
// Index 0 holds the empty string, which doubles as the name for any value
// outside the declared range.
var kindNames = [kindCount]string{
	KindUnknown:          "KindUnknown",
	KindGitHub:           "KindGitHub",
	KindGitHubEnterprise: "KindGitHubEnterprise",
	KindGitLab:           "KindGitLab",
	KindGitea:            "KindGitea",
	KindForgejo:          "KindForgejo",
	KindBitbucket:        "KindBitbucket",
	KindGoogleSource:     "KindGoogleSource",
}

// IsSelfHosted reports whether endpoints can be derived for the family.
//
// A self-hosted family derives its authorization and token endpoints from the
// Git remote root, so an operator does not have to register an application
// before the endpoints are known.
//
// Returns:
//   - bool: true for a self-hosted family.
func (k Kind) IsSelfHosted() bool {
	return k.profile().selfHosted
}

// LoginHintParam returns the authorization-URL query key for a username hint.
//
// Parameters:
//   - k: forge family.
//
// Returns:
//   - string: query key, or an empty string when the family takes no hint.
func (k Kind) LoginHintParam() string {
	return k.profile().loginHint
}

// NeedsClientSecret reports whether the family's application has a client
// secret that the browser grant must send.
//
// The device grant is not covered: GitHub accepts it with the client ID alone.
//
// Returns:
//   - bool: true for GitHub, GitHub Enterprise, Bitbucket, and Google Source.
func (k Kind) NeedsClientSecret() bool {
	return k.profile().secret
}

// String returns the identifier name of the kind.
//
// The name matches the Go constant, so a log line and a debugger agree.
//
// Returns:
//   - string: constant name, or the numeric form for an undeclared value.
func (k Kind) String() string {
	if k < 0 || int(k) >= kindCount {
		return "Kind(" + strconv.Itoa(int(k)) + ")"
	}

	return kindNames[k]
}

// Sum returns one past the largest declared kind.
//
// It exists so the checksum linter can verify that every constant added to
// this file is accounted for.
//
// Returns:
//   - int: exclusive upper bound on every declared kind.
func (k Kind) Sum() int {
	return kindCount
}

// SupportsBearerToken reports whether the family accepts a Bearer credential.
//
// This is the kind half of the answer. A host or realm can also enable it, so
// the caller has to consult those before deciding.
//
// Returns:
//   - bool: true when tokens for this family can be sent as Bearer.
func (k Kind) SupportsBearerToken() bool {
	return k.profile().bearer
}

// SupportsDeviceFlow reports whether the family implements the device grant.
//
// Returns:
//   - bool: true for GitLab and GitHub Enterprise.
func (k Kind) SupportsDeviceFlow() bool {
	return k.profile().device
}

// UsesGiteaLayout reports whether the family follows the Gitea endpoint paths
// and colon-separated scope names.
//
// Returns:
//   - bool: true for Gitea and Forgejo.
func (k Kind) UsesGiteaLayout() bool {
	return k.profile().giteaLayout
}

// UsesPublicClientID reports whether the family's servers register an
// application for this helper themselves.
//
// An instance of such a family can start an authorization before an operator
// has registered anything.
//
// Returns:
//   - bool: true for Gitea and Forgejo.
func (k Kind) UsesPublicClientID() bool {
	return k.profile().publicClientID
}

// profile returns the facts recorded for k.
//
// An out-of-range kind yields the zero profile, so an undeclared value behaves
// like KindUnknown rather than panicking.
//
// Parameters:
//   - k: forge family.
//
// Returns:
//   - kindProfile: recorded facts for k.
func (k Kind) profile() kindProfile {
	if k < 0 || int(k) >= kindCount {
		return kindProfile{}
	}

	return kindProfiles[k]
}
