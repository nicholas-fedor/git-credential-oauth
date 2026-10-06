// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/oauth2"
)

// wantProfile records the expected facts for one family.
//
// The table is written out rather than derived from the code, so a family that
// changes behavior fails this test instead of silently changing with it.
type wantProfile struct {
	loginHint      string
	kind           Kind
	selfHosted     bool
	registration   setupHint
	bearer         bool
	publicClientID bool
	giteaLayout    bool
	device         bool
	secret         bool
}

// allKinds is the authoritative list of families.
var allKinds = []Kind{
	KindUnknown,
	KindGitHub,
	KindGitHubEnterprise,
	KindGitLab,
	KindGitea,
	KindForgejo,
	KindBitbucket,
	KindGoogleSource,
}

// TestKindProfileTableCoversEveryFamily checks the table is exhaustive.
//
// Adding a family means adding a constant and a profile entry. A missing entry
// would fall back to the zero profile and quietly behave like KindUnknown, so
// this test fails when the two lists drift apart.
func TestKindProfileTableCoversEveryFamily(t *testing.T) {
	t.Parallel()

	assert.Len(t, allKinds, kindCount, "every kind is covered by this test")
	assert.Len(t, kindProfiles, kindCount, "table is sized to every kind")

	for kind := KindUnknown; int(kind) < kindCount; kind++ {
		assert.Contains(t, allKinds, kind, "kind %s is covered", kind)
	}
}

// TestKindProfileFacts pins each family's recorded behavior.
func TestKindProfileFacts(t *testing.T) {
	t.Parallel()

	want := []wantProfile{
		{
			kind:         KindUnknown,
			loginHint:    "",
			registration: setupRegister,
		},
		{
			kind:         KindGitHub,
			loginHint:    loginParamGitHub,
			registration: setupGitHub,
			secret:       true,
		},
		{
			kind:         KindGitHubEnterprise,
			loginHint:    loginParamGitHub,
			registration: setupGitHubEnterprise,
			selfHosted:   true,
			device:       true,
			secret:       true,
		},
		{
			kind:         KindGitLab,
			registration: setupGitLab,
			selfHosted:   true,
			device:       true,
		},
		{
			kind:           KindGitea,
			bearer:         true,
			publicClientID: true,
			giteaLayout:    true,
			selfHosted:     true,
		},
		{
			kind:           KindForgejo,
			bearer:         true,
			publicClientID: true,
			giteaLayout:    true,
			selfHosted:     true,
		},
		{
			kind:         KindBitbucket,
			registration: setupBitbucket,
			secret:       true,
		},
		{
			kind:         KindGoogleSource,
			loginHint:    loginParamGoogle,
			registration: setupGoogle,
			bearer:       true,
			secret:       true,
		},
	}

	require.Len(t, want, len(allKinds))

	for _, expected := range want {
		t.Run(expected.kind.String(), func(t *testing.T) {
			t.Parallel()

			profile := expected.kind.profile()

			assert.Equal(t, expected.selfHosted, expected.kind.IsSelfHosted(), "self-hosted")
			assert.Equal(t, expected.registration, profile.setup, "setup guidance")
			assert.Equal(t, expected.bearer, expected.kind.SupportsBearerToken(), "bearer")
			assert.Equal(t, expected.publicClientID, expected.kind.UsesPublicClientID(), "public client ID")
			assert.Equal(t, expected.giteaLayout, expected.kind.UsesGiteaLayout(), "gitea layout")
			assert.Equal(t, expected.device, expected.kind.SupportsDeviceFlow(), "device grant")
			assert.Equal(t, expected.loginHint, expected.kind.LoginHintParam(), "login hint")
			assert.Equal(t, expected.secret, expected.kind.NeedsClientSecret(), "client secret")
		})
	}
}

// TestSelfHostedFamiliesAreExactlyTheVendorOnes documents the split.
//
// Only these four derive their endpoints from a remote root. Adding another
// changes which hosts can work without a registered application.
func TestSelfHostedFamiliesAreExactlyTheVendorOnes(t *testing.T) {
	t.Parallel()

	var selfHosted []Kind

	for _, kind := range allKinds {
		if kind.IsSelfHosted() {
			selfHosted = append(selfHosted, kind)
		}
	}

	assert.Equal(t, []Kind{
		KindGitHubEnterprise,
		KindGitLab,
		KindGitea,
		KindForgejo,
	}, selfHosted)
}

// TestPublicClientIDFamiliesShareTheGiteaApplication checks the linkage.
//
// A family whose servers register an application for this helper must also use
// the Gitea layout, because that application only exists on those servers.
func TestPublicClientIDFamiliesShareTheGiteaApplication(t *testing.T) {
	t.Parallel()

	for _, kind := range allKinds {
		if kind.UsesPublicClientID() {
			assert.True(t, kind.UsesGiteaLayout(),
				"%s publishes a client ID but does not use the Gitea endpoints", kind)
		}
	}
}

// TestDeviceFlowFamiliesHaveADeviceEndpoint checks the derived endpoint.
//
// Advertising the grant without a device endpoint would make a user request
// device flow and then fail at the token step.
func TestDeviceFlowFamiliesHaveADeviceEndpoint(t *testing.T) {
	t.Parallel()

	for _, kind := range allKinds {
		if !kind.SupportsDeviceFlow() {
			continue
		}

		client, err := Derive(kind, "https://git.example.com")
		require.NoError(t, err, "kind %s", kind)
		assert.NotEmpty(
			t, client.Endpoint.DeviceAuthURL,
			"kind %s advertises a missing device endpoint", kind,
		)
	}
}

// TestFamiliesWithoutDeviceFlowDeriveNoDeviceEndpoint covers the negative case.
func TestFamiliesWithoutDeviceFlowDeriveNoDeviceEndpoint(t *testing.T) {
	t.Parallel()

	for _, kind := range allKinds {
		if kind.SupportsDeviceFlow() || !kind.IsSelfHosted() {
			continue
		}

		client, err := Derive(kind, "https://git.example.com")
		require.NoError(t, err, "kind %s", kind)
		assert.Empty(t, client.Endpoint.DeviceAuthURL, "kind %s must not offer a device endpoint", kind)
	}
}

// TestGiteaLayoutUsesParamsAuthStyle checks where credentials are placed.
//
// A Gitea instance rejects client credentials in the query string, so this is
// a compatibility requirement rather than a preference.
func TestGiteaLayoutUsesParamsAuthStyle(t *testing.T) {
	t.Parallel()

	for _, kind := range allKinds {
		if !kind.UsesGiteaLayout() {
			continue
		}

		client, err := Derive(kind, "https://git.example.com")
		require.NoError(t, err)
		assert.Equal(t, oauth2.AuthStyleInParams, client.Endpoint.AuthStyle, "kind %s", kind)
	}
}

// TestForgejoRequestsColonScopes checks the scope shape difference.
//
// Gitea takes no scope list here, while Forgejo needs its colon-separated
// repository scopes or the token comes back without repository access.
func TestForgejoRequestsColonScopes(t *testing.T) {
	t.Parallel()

	forgejo, err := Derive(KindForgejo, "https://git.example.com")
	require.NoError(t, err)
	assert.Equal(t, []string{"read:repository", "write:repository"}, forgejo.Scopes)

	gitea, err := Derive(KindGitea, "https://git.example.com")
	require.NoError(t, err)
	assert.Empty(t, gitea.Scopes)
}

// TestVendorFamiliesLeaveClientIDForTheOperator checks they cannot self-start.
//
// Neither GitLab nor GitHub Enterprise has a published application, so the
// derived client has to be completed from Git config before it can be used.
func TestVendorFamiliesLeaveClientIDForTheOperator(t *testing.T) {
	t.Parallel()

	for _, kind := range []Kind{KindGitLab, KindGitHubEnterprise} {
		client, err := Derive(kind, "https://git.example.com")
		require.NoError(t, err)

		assert.Empty(t, client.ClientID, "kind %s", kind)
		assert.Empty(t, client.ClientSecret, "kind %s", kind)
		assert.True(t, client.PKCE, "kind %s", kind)
	}
}

// TestSetupHintSaysWhereToRegisterAndWhatToSet pins every family's guidance.
//
// This program ships no application, so the hint is how a first-time user
// learns to register one. It is written out in full for each family because a
// wrong form path or a missing key leaves them with nothing else to go on.
func TestSetupHintSaysWhereToRegisterAndWhatToSet(t *testing.T) {
	t.Parallel()

	const (
		host   = "git.example.com"
		gitURL = "https://git.example.com"

		clientKey = "credential.https://git.example.com.oauthClientId"
		secretKey = "credential.https://git.example.com.oauthClientSecret"
		urlKeys   = "credential.https://git.example.com.oauthAuthURL, and " +
			"credential.https://git.example.com.oauthTokenURL"

		generic = "Register an OAuth application on this host with redirect URI " +
			"http://127.0.0.1."
		idAndSecret = " Then set Git config keys " + clientKey + " and " + secretKey + "."
	)

	want := map[Kind]string{
		KindUnknown: generic + " Then set Git config key " + clientKey + ".",
		KindGitHub: "Register an OAuth application at " +
			"https://github.com/settings/applications/new with callback URL " +
			"http://127.0.0.1." + idAndSecret,
		KindGitHubEnterprise: "Register an OAuth application at " +
			"https://git.example.com/settings/applications/new with callback URL " +
			"http://127.0.0.1." + idAndSecret,
		KindGitLab: "Register an OAuth application at " +
			"https://git.example.com/-/user_settings/applications with redirect URI " +
			"http://127.0.0.1, Confidential cleared, and the read_repository and " +
			"write_repository scopes. Then set Git config key " + clientKey + ".",
		KindGitea: generic + " Then set Git config keys " + clientKey + ", " +
			urlKeys + ".",
		KindForgejo: generic + " Then set Git config keys " + clientKey + ", " +
			urlKeys + ".",
		KindBitbucket: "Add an OAuth consumer under your Bitbucket workspace " +
			"settings with callback URL http://127.0.0.1 and repository read and " +
			"write permissions." + idAndSecret,
		KindGoogleSource: "Create an OAuth client ID of type Desktop app at " +
			"https://console.developers.google.com/auth/clients." + idAndSecret,
	}

	require.Len(t, want, len(allKinds), "every family has pinned guidance")

	for _, kind := range allKinds {
		detected := Detected{Host: host, Kind: kind}

		assert.Equal(t, want[kind], detected.SetupHint(gitURL), "kind %s", kind)
	}
}

// TestSetupHintNamesNoOtherProject checks the guidance stands on its own.
//
// A hint that linked to another project's issue tracker would send this
// program's users there for support.
func TestSetupHintNamesNoOtherProject(t *testing.T) {
	t.Parallel()

	for _, kind := range allKinds {
		detected := Detected{Host: "git.example.com", Kind: kind}
		hint := detected.SetupHint("https://git.example.com")

		assert.NotContains(t, hint, "/issues/", "kind %s", kind)
	}
}

// TestSecretFamiliesNameTheSecretKey checks the key hint follows the profile.
//
// A family whose token endpoint expects the application's secret has to say
// so, or a user sets the client ID alone and fails after the consent screen.
func TestSecretFamiliesNameTheSecretKey(t *testing.T) {
	t.Parallel()

	const gitURL = "https://git.example.com"

	for _, kind := range allKinds {
		detected := Detected{Host: "git.example.com", Kind: kind}
		hint := detected.ConfigKeyHint(gitURL)

		assert.Contains(t, hint, "oauthClientId", "kind %s", kind)

		if kind.NeedsClientSecret() {
			assert.Contains(t, hint, "oauthClientSecret", "kind %s", kind)
		} else {
			assert.NotContains(t, hint, "oauthClientSecret", "kind %s", kind)
		}
	}
}

// TestUndeclaredKindBehavesAsUnknown checks the table bounds.
//
// An out-of-range kind must degrade rather than panic, so a bad value read
// from configuration cannot crash a credential request.
func TestUndeclaredKindBehavesAsUnknown(t *testing.T) {
	t.Parallel()

	unknown := KindUnknown.profile()

	for _, kind := range []Kind{Kind(-1), Kind(kindCount), Kind(kindCount + 10)} {
		assert.Equal(t, unknown, kind.profile(), "kind %s", kind)
		assert.False(t, kind.IsSelfHosted())
		assert.False(t, kind.SupportsBearerToken())
		assert.False(t, kind.UsesGiteaLayout())
		assert.False(t, kind.UsesPublicClientID())
		assert.False(t, kind.SupportsDeviceFlow())
		assert.False(t, kind.NeedsClientSecret())
		assert.Empty(t, kind.LoginHintParam())
		assert.NotEmpty(t, kind.String(), "an undeclared kind still names itself")

		_, err := Derive(kind, "https://git.example.com")
		require.ErrorIs(t, err, errNotSelfHosted)
	}
}

// TestKindIsNeverEmpty checks the zero value is a real classification.
func TestKindIsNeverEmpty(t *testing.T) {
	t.Parallel()

	assert.Equal(t, KindUnknown, Kind(0))
	assert.True(t, slices.Contains(allKinds, Kind(0)))
}
