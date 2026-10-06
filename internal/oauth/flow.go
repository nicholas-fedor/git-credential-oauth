// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/oauth2"
)

// AuthCodeRequestURL builds an authorization-code URL with PKCE S256.
//
// suffix is appended verbatim. Pass a leading ampersand when adding extra
// query parameters. PKCE is always S256; callers do not choose the method.
//
// Parameters:
//   - cfg: client configuration. RedirectURL is included when set.
//   - state: CSRF state. Empty omits the parameter.
//   - verifier: PKCE code verifier.
//   - suffix: raw suffix appended to the URL.
//
// Returns:
//   - string: authorization URL.
func AuthCodeRequestURL(
	cfg oauth2.Config,
	state, verifier, suffix string,
) string {
	return cfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)) + suffix
}

// Acquire obtains a token, trying refresh before an interactive grant.
//
// Refresh failure is logged and does not fail the call. Device without
// provider support returns [ErrDeviceUnsupported] without contacting the
// authorization server. Otherwise the authorization-code grant runs with
// mandatory PKCE S256.
//
// Parameters:
//   - ctx: cancellation and deadline.
//   - cfg: client and endpoint configuration.
//   - in: grant selection and refresh token.
//
// Returns:
//   - Token: acquired credentials.
//   - error: non-nil when no grant succeeds.
func (a *ConfigAcquirer) Acquire(
	ctx context.Context,
	cfg oauth2.Config,
	in Input,
) (Token, error) {
	a.logger().Debug(ctx, "acquiring token",
		"refresh", in.RefreshToken != "",
		"device", in.Device,
		"device_flow", in.DeviceFlow,
		"pkce", in.PKCE,
	)

	if in.RefreshToken != "" {
		token, err := a.refresh(ctx, cfg, in.RefreshToken)
		if err == nil {
			return token, nil
		}

		a.logger().Warn(ctx, "refresh failed", "err", err)
	}

	if in.Device {
		return a.acquireDevice(ctx, cfg, in)
	}

	return a.acquireCode(ctx, cfg, in)
}

// refresh exchanges a refresh token for a new access token.
//
// A provider that rotates refresh tokens returns a new one here, and the value
// that was sent is then spent. FromRefresh records which happened so the caller
// can discard a stored token that can no longer be exchanged.
//
// Parameters:
//   - ctx: cancellation and deadline.
//   - cfg: OAuth client configuration.
//   - refreshToken: refresh token from a previous grant.
//
// Returns:
//   - Token: refreshed access token.
//   - error: token endpoint failure.
func (a *ConfigAcquirer) refresh(
	ctx context.Context,
	cfg oauth2.Config,
	refreshToken string,
) (Token, error) {
	source := cfg.TokenSource(withDoer(ctx, a.Doer), &oauth2.Token{
		RefreshToken: refreshToken,
	})

	token, err := source.Token()
	if err != nil {
		return Token{}, fmt.Errorf("refresh token: %w", err)
	}

	return FromRefresh(token, refreshToken), nil
}

// acquireCode runs the authorization-code grant with PKCE S256.
//
// The callback is always closed. Browser and callback errors are returned.
//
// Parameters:
//   - ctx: cancellation and deadline.
//   - cfg: OAuth client configuration.
//   - in: grant selection, including the auth URL suffix.
//
// Returns:
//   - Token: access token from the code exchange.
//   - error: missing browser or callback, state mismatch, or exchange failure.
func (a *ConfigAcquirer) acquireCode(
	ctx context.Context,
	cfg oauth2.Config,
	in Input,
) (Token, error) {
	if a == nil || a.Browser == nil {
		return Token{}, errNilBrowser
	}

	if a.Callback == nil {
		return Token{}, errNilCallback
	}

	state, err := newState(a.Nonce)
	if err != nil {
		return Token{}, err
	}

	verifier := oauth2.GenerateVerifier()

	callback, err := startCodeCallback(ctx, a.Callback, cfg.RedirectURL)
	if err != nil {
		return Token{}, err
	}

	resolved, err := openCodeBrowser(
		ctx,
		a.Browser,
		cfg,
		callback,
		state,
		verifier,
		in.AuthURLSuffix,
	)
	if err != nil {
		return Token{}, errors.Join(err, closeCallback(callback))
	}

	code, err := waitCodeQuery(ctx, callback, state)
	if err != nil {
		return Token{}, errors.Join(err, closeCallback(callback))
	}

	token, err := exchangeCode(ctx, resolved, a.Doer, code, verifier)

	return token, errors.Join(err, closeCallback(callback))
}

// startCodeCallback starts the loopback listener.
//
// A nil listener is an error. The caller closes a non-nil callback.
//
// Parameters:
//   - ctx: cancellation and deadline.
//   - factory: callback listener factory.
//   - redirectURL: configured redirect URI.
//
// Returns:
//   - Callback: listening callback.
//   - error: start failure or a nil callback.
//
//nolint:ireturn // CallbackFactory.Start returns the consumer interface.
func startCodeCallback(
	ctx context.Context,
	factory CallbackFactory,
	redirectURL string,
) (Callback, error) {
	callback, err := factory.Start(ctx, redirectURL)
	if err != nil {
		return nil, fmt.Errorf("start callback: %w", err)
	}

	if callback == nil {
		return nil, errNilCallback
	}

	return callback, nil
}

// openCodeBrowser opens the authorization URL and returns the resolved config.
//
// The listener URL replaces a configured zero-port redirect.
//
// Parameters:
//   - ctx: cancellation and deadline.
//   - browser: opener for the authorization URL.
//   - cfg: OAuth client configuration.
//   - callback: listener whose RedirectURL replaces the configured redirect.
//   - state: CSRF state.
//   - verifier: PKCE verifier.
//   - suffix: extra authorization URL query text.
//
// Returns:
//   - oauth2.Config: config with the resolved redirect URL.
//   - error: browser open failure.
func openCodeBrowser(
	ctx context.Context,
	browser Browser,
	cfg oauth2.Config,
	callback Callback,
	state string,
	verifier string,
	suffix string,
) (oauth2.Config, error) {
	// Use the resolved listener URL, not the configured zero-port URL.
	resolved := cfg

	resolved.RedirectURL = callback.RedirectURL()

	authURL := AuthCodeRequestURL(resolved, state, verifier, suffix)

	err := browser.Open(ctx, authURL)
	if err != nil {
		return oauth2.Config{}, fmt.Errorf("open browser: %w", err)
	}

	return resolved, nil
}

// waitCodeQuery waits for the redirect and checks state, error, and code.
//
// Parameters:
//   - ctx: cancellation and deadline.
//   - callback: listener waiting for the redirect.
//   - state: expected CSRF state.
//
// Returns:
//   - string: authorization code.
//   - error: wait failure, state mismatch, provider error, or a missing code.
func waitCodeQuery(
	ctx context.Context,
	callback Callback,
	state string,
) (string, error) {
	query, err := callback.Wait(ctx)
	if err != nil {
		return "", fmt.Errorf("wait for callback: %w", err)
	}

	if query.Get("state") != state {
		return "", ErrStateMismatch
	}

	authErr := query.Get("error")
	if authErr != "" {
		return "", fmt.Errorf("%w: %s", errAuthorization, authErr)
	}

	code := query.Get("code")
	if code == "" {
		return "", errMissingCode
	}

	return code, nil
}

// exchangeCode trades the authorization code for a token.
//
// Parameters:
//   - ctx: cancellation and deadline.
//   - cfg: OAuth client configuration with the resolved redirect URL.
//   - doer: HTTP client. Nil uses the default client.
//   - code: authorization code.
//   - verifier: PKCE verifier.
//
// Returns:
//   - Token: access token from the token endpoint.
//   - error: exchange failure.
func exchangeCode(
	ctx context.Context,
	cfg oauth2.Config,
	doer Doer,
	code string,
	verifier string,
) (Token, error) {
	exchanged, err := cfg.Exchange(
		withDoer(ctx, doer),
		code,
		oauth2.VerifierOption(verifier),
	)
	if err != nil {
		return Token{}, fmt.Errorf("exchange code: %w", err)
	}

	return FromOAuth2(exchanged), nil
}
