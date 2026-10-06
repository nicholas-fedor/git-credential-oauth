// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/nicholas-fedor/git-credential-oauth/internal/credential"
	"github.com/nicholas-fedor/git-credential-oauth/internal/forge"
	"github.com/nicholas-fedor/git-credential-oauth/internal/oauth"
)

// configOverrides holds git config values already read for one URL.
type configOverrides struct {
	ClientID      string
	ClientSecret  string
	Scopes        string
	AuthURL       string
	TokenURL      string
	DeviceAuthURL string
	RedirectURL   string
	ExpiryMargin  string

	ClientIDFound      bool
	ClientSecretFound  bool
	ScopesFound        bool
	AuthURLFound       bool
	TokenURLFound      bool
	DeviceAuthURLFound bool
	RedirectURLFound   bool
	ExpiryMarginFound  bool
}

// applied is a client after git config overrides.
type applied struct {
	RedirectURL string
	Client      forge.Client
}

// resolution is a pure OAuth client ready for acquisition or a config error.
type resolution struct {
	Config        oauth2.Config
	GitURL        string
	MarginRaw     string
	Detected      forge.Detected
	Input         oauth.Input
	Scopes        []string
	Client        forge.Client
	Margin        time.Duration
	MarginInvalid bool
}

const (
	// defaultProtocol is used when the credential request omits protocol.
	// An https remote is the only supported transport, so this matches
	// schemeHTTPS rather than naming a second literal.
	defaultProtocol = schemeHTTPS

	// usernamePlaceholderOAuth2 is the forge placeholder username. Forwarding
	// it as a login hint would send the placeholder to the provider in place of
	// the operator.s real account.
	usernamePlaceholderOAuth2 = "oauth2"

	// scopeReadRepository is a GitLab-shaped scope name.
	scopeReadRepository = "read_repository"

	// scopeWriteRepository is a GitLab-shaped scope name.
	scopeWriteRepository = "write_repository"

	// keyJoinCount is the key count that gets "and" between the two names.
	keyJoinCount = 2
)

// resolve builds an OAuth config without reading config or acquiring a token.
//
// A derived self-hosted client is refined with published RFC 8414 metadata
// when the provider offers it. Incomplete clients return a missing-config error
// whose hint says where to register an application and which keys record it.
//
// Parameters:
//   - ctx: cancellation and request scope, used for metadata discovery.
//   - reg: forge registry. Nil is an error.
//   - req: git credential request.
//   - opts: bearer and device-flow options.
//   - overrides: git config values already read for the URL.
//
// Returns:
//   - resolution: OAuth config, detected host, and expiry margin.
//   - error: nil registry, derive failure, bad endpoint URL, or missing config.
func resolve(
	ctx context.Context,
	reg forge.Registry,
	req credential.Request,
	opts Options,
	overrides configOverrides,
) (resolution, error) {
	if reg == nil {
		return resolution{}, errNilRegistry
	}

	gitURL := protocolURL(req.Protocol, req.Host)
	detected := forge.Detect(reg, req.Host, strings.Join(req.WWWAuth, "\n"))

	client, redirect, err := resolveClient(
		ctx, reg, detected, req.Host, gitURL, overrides,
	)
	if err != nil {
		return resolution{}, err
	}

	partial := newResolution(detected, client, gitURL, overrides)

	if configIncomplete(client) {
		return partial, &MissingConfigError{
			Host:   req.Host,
			GitURL: gitURL,
			Hint:   missingConfigHint(detected, client, gitURL),
		}
	}

	partial.Config = oauthConfig(client, redirect)
	partial.Input = oauthInput(req, detected, client, opts)

	return partial, nil
}

// resolveClient produces the client and redirect the grant will use.
//
// Lookup, derivation, metadata discovery, and operator overrides are applied
// in that order, because each step refines the one before it: a derived client
// has no operator values yet, and an operator key has to win over everything
// this program knows.
//
// Parameters:
//   - ctx: cancellation and request scope.
//   - reg: forge registry.
//   - detected: host classification.
//   - host: Git hostname.
//   - gitURL: absolute Git remote URL.
//   - overrides: git config values already read for the URL.
//
// Returns:
//   - client: forge client with every applied change.
//   - redirect: loopback redirect, empty for an ephemeral port.
//   - error: derive failure or a bad configured endpoint.
func resolveClient(
	ctx context.Context,
	reg forge.Registry,
	detected forge.Detected,
	host string,
	gitURL string,
	overrides configOverrides,
) (forge.Client, string, error) {
	client, registered, err := lookupOrDerive(reg, detected, host, gitURL)
	if err != nil {
		return forge.Client{}, "", err
	}

	if !registered {
		client = discoverEndpoints(ctx, detected.Kind, host, client)
	}

	overridden, err := applyOverrides(client, gitURL, overrides)
	if err != nil {
		return forge.Client{}, "", err
	}

	return overridden.Client, overridden.RedirectURL, nil
}

// newResolution collects the findings resolve reports on every path.
//
// Warnings are attached before the error is checked so the caller can log what
// it learned about an operator's configuration even when the request fails.
//
// Parameters:
//   - detected: host classification.
//   - client: resolved client.
//   - gitURL: absolute Git remote URL.
//   - overrides: git config values already read for the URL.
//
// Returns:
//   - resolution: partial resolution with no OAuth config or grant input.
func newResolution(
	detected forge.Detected,
	client forge.Client,
	gitURL string,
	overrides configOverrides,
) resolution {
	margin, marginUsable := parsedMargin(overrides)

	return resolution{
		Detected:      detected,
		Client:        client,
		Config:        oauth2.Config{},
		Input:         oauth.Input{},
		GitURL:        gitURL,
		MarginRaw:     overrides.ExpiryMargin,
		Scopes:        slices.Clone(client.Scopes),
		Margin:        margin,
		MarginInvalid: overrides.ExpiryMarginFound && !marginUsable,
	}
}

// protocolURL joins protocol and host, defaulting an empty protocol to https.
//
// Parameters:
//   - protocol: URL scheme. Empty becomes https.
//   - host: Git hostname.
//
// Returns:
//   - string: scheme://host.
func protocolURL(protocol, host string) string {
	if protocol == "" {
		protocol = defaultProtocol
	}

	return protocol + "://" + host
}

// lookupOrDerive returns a registry client or a derived self-hosted client.
//
// A registry miss on a non-self-hosted kind returns a zero client.
//
// Parameters:
//   - reg: forge registry.
//   - detected: host classification.
//   - host: Git hostname.
//   - gitURL: absolute Git remote root used to derive endpoints.
//
// Returns:
//   - forge.Client: registry or derived client.
//   - registered: true when the client came from the registry rather than from
//     derivation. Only a derived client is worth refining with published
//     metadata, because a registry entry is this program's own decision.
//   - error: derive failure.
func lookupOrDerive(
	reg forge.Registry,
	detected forge.Detected,
	host string,
	gitURL string,
) (forge.Client, bool, error) {
	client, registered := reg.Lookup(host)
	if registered || !reg.SelfHosted(detected.Kind) {
		return client, registered, nil
	}

	derived, err := forge.Derive(detected.Kind, gitURL)
	if err != nil {
		return forge.Client{}, false, fmt.Errorf("derive oauth client: %w", err)
	}

	return derived, false, nil
}

// discoverEndpoints replaces derived endpoints with published ones.
//
// RFC 9700 section 2.6 recommends using RFC 8414 metadata when a provider
// publishes it, naming misconfigured endpoint URLs as the reason: an endpoint
// recorded for the wrong host belongs to somebody else. A self-hosted instance
// that moved its endpoints is the common case, and only the provider knows
// where they went.
//
// Every failure falls back to the derived client. A host that publishes
// nothing, is offline, or serves a document naming somebody else must not stop
// a credential that would otherwise work.
//
// Parameters:
//   - ctx: cancellation and deadline for the fetch.
//   - kind: forge family.
//   - host: Git hostname.
//   - derived: client to refine.
//
// Returns:
//   - forge.Client: published endpoints when usable, otherwise derived.
func discoverEndpoints(
	ctx context.Context,
	kind forge.Kind,
	host string,
	derived forge.Client,
) forge.Client {
	meta, found, err := forge.Discover(ctx, host, nil)
	if err != nil || !found {
		return derived
	}

	published, found, err := forge.FromMetadata(kind, host, meta)
	if err != nil || !found {
		return derived
	}

	if published.ClientID == "" {
		published.ClientID = derived.ClientID
	}

	if len(published.Scopes) == 0 {
		published.Scopes = derived.Scopes
	}

	return published
}

// applyOverrides copies client and applies found git config values.
//
// Scope values are split on whitespace. Endpoint overrides are resolved
// against gitURL.
//
// Parameters:
//   - client: base client.
//   - gitURL: absolute Git remote root used to resolve relative URLs.
//   - overrides: git config values. Unset keys are ignored.
//
// Returns:
//   - applied: updated client and redirect URL.
//   - error: an endpoint URL could not be resolved.
func applyOverrides(
	client forge.Client,
	gitURL string,
	overrides configOverrides,
) (applied, error) {
	updated := client

	if overrides.ClientIDFound {
		updated.ClientID = overrides.ClientID
	}

	if overrides.ClientSecretFound {
		updated.ClientSecret = overrides.ClientSecret
	}

	if overrides.ScopesFound {
		updated.Scopes = strings.Fields(overrides.Scopes)
	}

	updated, err := applyEndpointOverrides(updated, gitURL, overrides)
	if err != nil {
		return applied{}, err
	}

	redirect := ""
	if overrides.RedirectURLFound {
		redirect = overrides.RedirectURL
	}

	return applied{Client: updated, RedirectURL: redirect}, nil
}

// applyEndpointOverrides resolves found endpoint URLs against gitURL.
//
// An empty configured value clears that URL.
//
// Parameters:
//   - client: client whose endpoint is copied and updated.
//   - gitURL: absolute base URL for relative references.
//   - overrides: auth, token, and device URL overrides.
//
// Returns:
//   - forge.Client: client with updated endpoint URLs.
//   - error: a configured URL could not be resolved.
func applyEndpointOverrides(
	client forge.Client,
	gitURL string,
	overrides configOverrides,
) (forge.Client, error) {
	updated := client

	if overrides.AuthURLFound {
		resolved, err := resolveConfiguredURL(gitURL, overrides.AuthURL)
		if err != nil {
			return forge.Client{}, err
		}

		updated.Endpoint.AuthURL = resolved
	}

	if overrides.TokenURLFound {
		resolved, err := resolveConfiguredURL(gitURL, overrides.TokenURL)
		if err != nil {
			return forge.Client{}, err
		}

		updated.Endpoint.TokenURL = resolved
	}

	if overrides.DeviceAuthURLFound {
		resolved, err := resolveConfiguredURL(gitURL, overrides.DeviceAuthURL)
		if err != nil {
			return forge.Client{}, err
		}

		updated.Endpoint.DeviceAuthURL = resolved
	}

	return updated, nil
}

// resolveConfiguredURL resolves a non-empty ref, and treats empty as a clear.
//
// Parameters:
//   - gitURL: absolute base URL.
//   - value: configured URL. Empty clears the field.
//
// Returns:
//   - string: resolved URL, or empty when value is empty.
//   - error: value could not be resolved against gitURL.
func resolveConfiguredURL(gitURL, value string) (string, error) {
	if value == "" {
		return "", nil
	}

	resolved, err := resolveReference(gitURL, value)
	if err != nil {
		return "", err
	}

	return resolved, nil
}

// resolveReference resolves ref against baseURL.
//
// Absolute refs are returned unchanged. Relative refs join the git URL. A
// cleartext http result is rejected: these values are the endpoints an
// authorization code and an access token are sent to, and RFC 9700 section 2.6
// forbids transmitting them over an unencrypted connection. A result with no
// host, or one whose string form does not parse back, is rejected too.
//
// Parameters:
//   - baseURL: absolute Git remote root.
//   - ref: absolute or relative URL.
//
// Returns:
//   - string: resolved https URL.
//   - error: baseURL or ref cannot be parsed, or the result is not an https
//     URL with a host.
func resolveReference(baseURL, ref string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse git url: %w", err)
	}

	parsed, err := url.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("parse oauth url %q: %w", ref, err)
	}

	resolved := base.ResolveReference(parsed)

	if resolved.Scheme != schemeHTTPS {
		return "", fmt.Errorf(
			"%w: %q resolves to %q",
			errInsecureEndpoint,
			ref,
			resolved.Scheme,
		)
	}

	result := resolved.String()

	_, reparseErr := url.Parse(result)
	if resolved.Host == "" || reparseErr != nil {
		return "", fmt.Errorf("%w: %q resolves to %q", errMalformedEndpoint, ref, result)
	}

	return result, nil
}

// parsedMargin returns the configured margin, or the default when unset or bad.
//
// The boolean reports whether the result is usable, which is true both when
// nothing was configured and when the configured value parsed. Only a value
// that was set and did not parse is unusable, and that is the one case the
// caller warns about.
//
// Parameters:
//   - overrides: git config values. ExpiryMargin is used when found.
//
// Returns:
//   - [time.Duration]: configured margin, or DefaultExpiryMargin.
//   - bool: false when a configured value could not be parsed.
func parsedMargin(overrides configOverrides) (time.Duration, bool) {
	if !overrides.ExpiryMarginFound {
		return DefaultExpiryMargin, true
	}

	return parseExpiryMargin(overrides.ExpiryMargin)
}

// parseExpiryMargin parses value and reports whether the parse succeeded.
//
// An unparseable value falls back to the default rather than failing the
// request, so a typo cannot lock a user out of their credential. The caller
// warns about it separately.
//
// Parameters:
//   - value: duration string accepted by [time.ParseDuration].
//
// Returns:
//   - [time.Duration]: parsed duration, or DefaultExpiryMargin on failure.
//   - bool: true when parsing succeeded.
func parseExpiryMargin(value string) (time.Duration, bool) {
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return DefaultExpiryMargin, false
	}

	return parsed, true
}

// configIncomplete reports whether the client cannot start an OAuth grant.
//
// Parameters:
//   - client: client after overrides.
//
// Returns:
//   - bool: true when the client ID or either endpoint URL is empty.
func configIncomplete(client forge.Client) bool {
	return client.ClientID == "" ||
		client.Endpoint.AuthURL == "" ||
		client.Endpoint.TokenURL == ""
}

// missingConfigHint selects operator guidance for an incomplete client.
//
// A client with no ID has no application behind it yet, and this program ships
// none, so the operator is told where to register one before the keys are
// named. A client that already has an ID only lacks endpoints, so naming those
// keys is the more useful guidance.
//
// Parameters:
//   - detected: host classification used to choose the hint.
//   - client: incomplete client.
//   - gitURL: Git remote URL inserted into the hint.
//
// Returns:
//   - string: operator guidance.
func missingConfigHint(
	detected forge.Detected,
	client forge.Client,
	gitURL string,
) string {
	if client.ClientID == "" {
		return detected.SetupHint(gitURL)
	}

	return ensureMentions(
		detected.ConfigKeyHint(gitURL),
		"oauthAuthURL",
		"oauthTokenURL",
	)
}

// ensureMentions appends key names that hint does not already contain.
//
// Parameters:
//   - hint: existing guidance sentence.
//   - keys: config key names to mention when absent.
//
// Returns:
//   - string: hint, with missing key names appended.
func ensureMentions(hint string, keys ...string) string {
	missing := make([]string, 0, len(keys))

	for _, key := range keys {
		if strings.Contains(hint, key) {
			continue
		}

		missing = append(missing, key)
	}

	if len(missing) == 0 {
		return hint
	}

	return hint + " Also set " + joinKeys(missing) + "."
}

// joinKeys renders one or two key names for a hint sentence.
//
// Parameters:
//   - keys: key names. Empty yields an empty string.
//
// Returns:
//   - string: one name, or two names joined with "and".
func joinKeys(keys []string) string {
	if len(keys) == 1 {
		return keys[0]
	}

	if len(keys) == keyJoinCount {
		return keys[0] + " and " + keys[1]
	}

	return strings.Join(keys, ", ")
}

// oauthConfig copies client fields into an oauth2 config.
//
// Endpoint is assigned whole so AuthStyle is preserved.
//
// Parameters:
//   - client: source client.
//   - redirectURL: OAuth redirect URL. Empty leaves it unset.
//
// Returns:
//   - oauth2.Config: copy of the client credentials and endpoint.
func oauthConfig(client forge.Client, redirectURL string) oauth2.Config {
	return oauth2.Config{
		ClientID:     client.ClientID,
		ClientSecret: client.ClientSecret,
		Endpoint:     client.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       slices.Clone(client.Scopes),
	}
}

// oauthInput passes grant selection through to the acquirer.
//
// Parameters:
//   - req: git credential request. The refresh token and username are used.
//   - detected: host classification. LoginHintParam names the auth URL
//     parameter.
//   - client: client whose device-flow and PKCE flags are copied.
//   - opts: device-flow option.
//
// Returns:
//   - oauth.Input: grant selection for Acquire.
func oauthInput(
	req credential.Request,
	detected forge.Detected,
	client forge.Client,
	opts Options,
) oauth.Input {
	return oauth.Input{
		RefreshToken:  req.OAuthRefreshToken,
		AuthURLSuffix: authURLSuffix(req.Username, detected.LoginHintParam()),
		Device:        opts.Device,
		DeviceFlow:    client.DeviceFlow,
		PKCE:          client.PKCE,
	}
}

// authURLSuffix appends a login hint when the forge names a parameter.
//
// An empty username, the oauth2 username, or an empty parameter yields no
// suffix.
//
// Parameters:
//   - username: git credential username.
//   - param: authorization URL parameter name, such as login.
//
// Returns:
//   - string: "&param=username", or empty.
func authURLSuffix(username, param string) string {
	if username == "" || username == usernamePlaceholderOAuth2 || param == "" {
		return ""
	}

	return "&" + param + "=" + username
}

// offendingScopes returns GitLab-shaped scopes on a Gitea or Forgejo host.
//
// Parameters:
//   - kind: forge family. Other kinds yield nil.
//   - scopes: configured scope names.
//
// Returns:
//   - []string: scopes that contain read_repository or write_repository.
func offendingScopes(kind forge.Kind, scopes []string) []string {
	if !giteaFamily(kind) {
		return nil
	}

	var matched []string

	for _, scope := range scopes {
		if gitlabShaped(scope) {
			matched = append(matched, scope)
		}
	}

	return matched
}

// giteaFamily reports whether kind uses Gitea-style scope names.
//
// The scope shape is a property of the forge family rather than of this
// package, so the answer comes from the family profile.
//
// Parameters:
//   - kind: forge family.
//
// Returns:
//   - bool: true for Gitea and Forgejo.
func giteaFamily(kind forge.Kind) bool {
	return kind.UsesGiteaLayout()
}

// gitlabShaped reports whether scope contains a GitLab repository scope name.
//
// Parameters:
//   - scope: one scope string.
//
// Returns:
//   - bool: true when scope contains read_repository or write_repository.
func gitlabShaped(scope string) bool {
	return strings.Contains(scope, scopeReadRepository) ||
		strings.Contains(scope, scopeWriteRepository)
}
