// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"time"

	"golang.org/x/oauth2"
)

// Token is an OAuth 2.0 access token and its refresh metadata.
type Token struct {
	// Expiry is when the access token stops being valid. Zero means the
	// provider issued no expiry, which is not the same as never expiring.
	Expiry time.Time

	// AccessToken is the bearer credential sent to the forge.
	AccessToken string

	// TokenType is the authorization scheme the token is sent with.
	TokenType string

	// RefreshToken is the token used to obtain the next access token.
	//
	// A provider that does not rotate refresh tokens returns no replacement in
	// its response, so the previous value has to be carried forward or the next
	// request would have nothing to refresh with.
	RefreshToken string
}

// HasExpired reports whether the access token is expired at now.
//
// A zero Expiry never expires. margin is a pre-expiry window: the token is
// expired when Expiry is at or before now+margin, including the exact
// boundary. A negative margin is treated as zero.
//
// Parameters:
//   - now: instant to compare against.
//   - margin: pre-expiry window.
//
// Returns:
//   - bool: true when the token is expired or inside the margin.
func (t Token) HasExpired(now time.Time, margin time.Duration) bool {
	if t.Expiry.IsZero() {
		return false
	}

	if margin < 0 {
		margin = 0
	}

	return !t.Expiry.After(now.Add(margin))
}

// FromOAuth2 copies an oauth2 token into a Token.
//
// A nil tok yields the zero Token. Extra oauth2 fields are dropped.
//
// Parameters:
//   - tok: source token. Nil is allowed.
//
// Returns:
//   - Token: copied credentials.
func FromOAuth2(tok *oauth2.Token) Token {
	if tok == nil {
		return Token{}
	}

	return Token{
		AccessToken:  tok.AccessToken,
		TokenType:    tok.TokenType,
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
	}
}

// FromRefresh copies the result of a refresh grant into a Token.
//
// A provider that rotates refresh tokens sends a new one and the previous value
// is then spent. A provider that does not rotate sends no refresh_token
// field at all, so the response has none to copy. Carrying the previous value
// forward is what keeps a non-rotating provider.s credential usable; without it
// every access token expiry would send the user back through an interactive
// authorization.
//
// Parameters:
//   - tok: token returned by the refresh exchange.
//   - previous: refresh token used for this exchange.
//
// Returns:
//   - Token: refreshed credentials.
func FromRefresh(tok *oauth2.Token, previous string) Token {
	token := FromOAuth2(tok)
	if token.RefreshToken == "" {
		token.RefreshToken = previous
	}

	return token
}
