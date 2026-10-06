// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nicholas-fedor/git-credential-oauth/internal/credential"
	"github.com/nicholas-fedor/git-credential-oauth/internal/forge"
	"github.com/nicholas-fedor/git-credential-oauth/internal/oauth"
)

// Deps are the collaborators for a credential get.
//
// Deps has no git runner. A credential request cannot mutate git config.
type Deps struct {
	// Config reads oauth git config keys. Nil means every key is unset.
	Config ConfigReader

	// Auth acquires the OAuth token.
	Auth oauth.Acquirer

	// Forges looks up and derives OAuth clients.
	Forges forge.Registry

	// Log receives diagnostic events. Nil is a no-op.
	Log Logger

	// Now supplies the current instant. Nil uses [time.Now].
	Now func() time.Time
}

// Service obtains an OAuth credential for a git request.
type Service struct {
	// Deps are the collaborators for Get.
	Deps Deps
}

// overrideSetter binds one git config key to the field it fills.
type overrideSetter struct {
	// apply records the trimmed value and marks the field as found.
	apply func(string)

	// key is the full git config key to read.
	key string
}

// DefaultExpiryMargin is subtracted from token expiry before git sees it.
const DefaultExpiryMargin = 2 * time.Minute

const (
	// keyClientID is the full git config key for the OAuth client ID.
	keyClientID = "credential.oauthClientId"

	// keyClientSecret is the full git config key for the OAuth client secret.
	keyClientSecret = "credential.oauthClientSecret"

	// keyScopes is the full git config key for OAuth scopes.
	keyScopes = "credential.oauthScopes"

	// keyAuthURL is the full git config key for the authorization URL.
	keyAuthURL = "credential.oauthAuthURL"

	// keyTokenURL is the full git config key for the token URL.
	keyTokenURL = "credential.oauthTokenURL"

	// keyDeviceAuthURL is the full git config key for the device authorization
	// URL.
	keyDeviceAuthURL = "credential.oauthDeviceAuthURL"

	// keyRedirectURL is the full git config key for the redirect URL.
	keyRedirectURL = "credential.oauthRedirectURL"

	// keyExpiryMargin is the full git config key for the expiry safety margin.
	keyExpiryMargin = "credential.oauthExpiryMargin"
)

// Get returns an ordered credential response for req.
//
// The work splits into four steps that each have their own failure mode: read
// the operator's git config, resolve it to an OAuth client, acquire a token,
// and emit the credential. Only the first and third can fail a request, which
// is why resolve's warnings are collected before its error is checked.
//
// Parameters:
//   - ctx: cancellation and deadline for config lookup and token acquisition.
//   - req: git credential request.
//   - opts: bearer and device-flow options.
//
// Returns:
//   - credential.Response: ordered credential attributes.
//   - error: missing configuration, config lookup failure, or token
//     acquisition failure.
func (s Service) Get(
	ctx context.Context,
	req credential.Request,
	opts Options,
) (credential.Response, error) {
	overrides, err := s.readOverrides(ctx, protocolURL(req.Protocol, req.Host))
	if err != nil {
		return credential.Response{}, err
	}

	resolved, err := resolve(ctx, s.Deps.Forges, req, opts, overrides)
	s.noteWarnings(ctx, &resolved)

	if err != nil {
		return credential.Response{}, err
	}

	token, err := s.acquire(ctx, &resolved)
	if err != nil {
		return credential.Response{}, err
	}

	return emit(req, resolved.Detected, token, opts, s.now(), resolved.Margin), nil
}

// acquire obtains a token for the resolved client.
//
// Parameters:
//   - ctx: cancellation and deadline for the grant.
//   - resolved: client, redirect, and grant input.
//
// Returns:
//   - oauth.Token: acquired token.
//   - error: nil-acquirer or token acquisition failure.
func (s Service) acquire(
	ctx context.Context,
	resolved *resolution,
) (oauth.Token, error) {
	if s.Deps.Auth == nil {
		return oauth.Token{}, errNilAcquirer
	}

	token, err := s.Deps.Auth.Acquire(ctx, resolved.Config, resolved.Input)
	if err != nil {
		return oauth.Token{}, wrapAcquire(err)
	}

	return token, nil
}

// noteWarnings logs non-fatal resolve findings.
//
// An invalid expiry margin and GitLab-shaped scopes on Gitea or Forgejo are
// warnings. Neither fails the request.
//
// Parameters:
//   - ctx: cancellation and request scope.
//   - resolved: resolved client and warning flags.
func (s Service) noteWarnings(ctx context.Context, resolved *resolution) {
	if resolved.MarginInvalid {
		warnInvalidMargin(ctx, s.Deps.Log, resolved.MarginRaw)
	}

	noteScopeShape(ctx, s.Deps.Log, resolved.Detected.Kind, resolved.Scopes)
}

// now returns Deps.Now or [time.Now] when Now is nil.
//
// Parameters:
//   - s: service whose clock is read.
//
// Returns:
//   - [time.Time]: injected clock, or the current time.
func (s Service) now() time.Time {
	if s.Deps.Now == nil {
		return time.Now()
	}

	return s.Deps.Now()
}

// readInto applies key when it is set.
//
// A config lookup error is returned. An unset key does not call apply.
//
// Parameters:
//   - ctx: cancellation and request scope.
//   - gitURL: Git remote URL matched by git config.
//   - key: full git config key.
//   - apply: called with the trimmed value when the key is set.
//
// Returns:
//   - error: config lookup failure.
func (s Service) readInto(
	ctx context.Context,
	gitURL string,
	key string,
	apply func(string),
) error {
	value, found, err := s.readKey(ctx, key, gitURL)
	if err != nil {
		return err
	}

	if found {
		apply(value)
	}

	return nil
}

// readKey returns one trimmed config value.
//
// A non-nil error is returned and is not treated as an unset key.
//
// Parameters:
//   - ctx: cancellation and request scope.
//   - key: full git config key.
//   - gitURL: Git remote URL matched by git config.
//
// Returns:
//   - string: trimmed value.
//   - bool: true when the key is set.
//   - error: config lookup failure, or a nil reader.
func (s Service) readKey(
	ctx context.Context,
	key string,
	gitURL string,
) (string, bool, error) {
	if s.Deps.Config == nil {
		return "", false, nil
	}

	value, found, err := s.Deps.Config.Get(ctx, key, gitURL)
	if err != nil {
		return "", false, fmt.Errorf("read %s: %w", key, err)
	}

	return strings.TrimSpace(value), found, nil
}

// readOverrides loads every oauth git config key for gitURL.
//
// Each key maps to a small setter rather than being read into a temporary, so
// adding a key means adding one table row instead of a field, a row, and a
// separate assignment.
//
// Parameters:
//   - ctx: cancellation and request scope.
//   - gitURL: Git remote URL matched by git config.
//
// Returns:
//   - configOverrides: values that were set.
//   - error: config lookup failure.
func (s Service) readOverrides(
	ctx context.Context,
	gitURL string,
) (configOverrides, error) {
	var overrides configOverrides

	for _, setter := range overrideSetters(&overrides) {
		err := s.readInto(ctx, gitURL, setter.key, setter.apply)
		if err != nil {
			return configOverrides{}, err
		}
	}

	return overrides, nil
}

// overrideSetters describes every key readOverrides loads.
//
// The order is the order the keys are read in, which is the order they are
// listed in the flag documentation.
//
// Parameters:
//   - overrides: destination that each setter writes to.
//
// Returns:
//   - []overrideSetter: one setter per configurable key.
func overrideSetters(overrides *configOverrides) []overrideSetter {
	return []overrideSetter{
		{key: keyClientID, apply: func(value string) {
			overrides.ClientID = value
			overrides.ClientIDFound = true
		}},
		{key: keyClientSecret, apply: func(value string) {
			overrides.ClientSecret = value
			overrides.ClientSecretFound = true
		}},
		{key: keyScopes, apply: func(value string) {
			overrides.Scopes = value
			overrides.ScopesFound = true
		}},
		{key: keyAuthURL, apply: func(value string) {
			overrides.AuthURL = value
			overrides.AuthURLFound = true
		}},
		{key: keyTokenURL, apply: func(value string) {
			overrides.TokenURL = value
			overrides.TokenURLFound = true
		}},
		{key: keyDeviceAuthURL, apply: func(value string) {
			overrides.DeviceAuthURL = value
			overrides.DeviceAuthURLFound = true
		}},
		{key: keyRedirectURL, apply: func(value string) {
			overrides.RedirectURL = value
			overrides.RedirectURLFound = true
		}},
		{key: keyExpiryMargin, apply: func(value string) {
			overrides.ExpiryMargin = value
			overrides.ExpiryMarginFound = true
		}},
	}
}

// warnInvalidMargin logs a duration that could not be parsed.
//
// A nil logger is ignored.
//
// Parameters:
//   - ctx: cancellation and request scope.
//   - log: destination. Nil skips the warning.
//   - raw: duration text that failed to parse.
func warnInvalidMargin(ctx context.Context, log Logger, raw string) {
	if log == nil {
		return
	}

	log.Warn(ctx, "invalid oauth expiry margin", "value", raw)
}

// noteScopeShape warns when a Gitea or Forgejo host uses a GitLab-shaped scope.
//
// The request is not failed.
//
// Parameters:
//   - ctx: cancellation and request scope.
//   - log: destination. Nil skips the warning.
//   - kind: forge family. Only Gitea and Forgejo warn.
//   - scopes: configured scope names.
func noteScopeShape(
	ctx context.Context,
	log Logger,
	kind forge.Kind,
	scopes []string,
) {
	matched := offendingScopes(kind, scopes)
	if log == nil || len(matched) == 0 {
		return
	}

	log.Warn(ctx, "gitlab-shaped scope on gitea or forgejo", "scopes", matched)
}

// wrapAcquire wraps a token acquisition error and preserves
// ErrDeviceUnsupported.
//
// Parameters:
//   - err: error from Acquire.
//
// Returns:
//   - error: wrapped acquisition error. ErrDeviceUnsupported still matches.
func wrapAcquire(err error) error {
	if errors.Is(err, oauth.ErrDeviceUnsupported) {
		return fmt.Errorf("device flow unsupported: %w", err)
	}

	return fmt.Errorf("acquire oauth token: %w", err)
}
