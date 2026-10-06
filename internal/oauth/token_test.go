// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"golang.org/x/oauth2"
)

// TestFromRefreshCarriesForwardANonRotatingToken is the regression this guards.
//
// Most providers do not rotate refresh tokens, and a token endpoint that does
// not rotate simply omits refresh_token from its response. Copying the response
// verbatim would empty the stored token, so the next request would have nothing
// to refresh with and the user would face an interactive authorization every
// time the access token expired.
func TestFromRefreshCarriesForwardANonRotatingToken(t *testing.T) {
	t.Parallel()

	const previous = "stored-refresh-token"

	got := FromRefresh(&oauth2.Token{
		AccessToken: "new-access",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}, previous)

	assert.Equal(t, previous, got.RefreshToken, "the stored token survived")
	assert.Equal(t, "new-access", got.AccessToken)
}

// TestFromRefreshUsesARotatedToken covers the other provider behavior.
//
// A rotating provider sends a replacement and the previous value is spent.
// RFC 6749 section 6 requires the new one to be used and the old discarded,
// which is what carrying the replacement through does.
func TestFromRefreshUsesARotatedToken(t *testing.T) {
	t.Parallel()

	got := FromRefresh(&oauth2.Token{
		AccessToken:  "new-access",
		RefreshToken: "rotated-refresh-token",
		TokenType:    "Bearer",
	}, "spent-refresh-token")

	assert.Equal(t, "rotated-refresh-token", got.RefreshToken)
}

// TestFromRefreshWithNoPreviousToken covers a grant that issued none.
func TestFromRefreshWithNoPreviousToken(t *testing.T) {
	t.Parallel()

	got := FromRefresh(&oauth2.Token{
		AccessToken: "new-access",
		TokenType:   "Bearer",
	}, "")

	assert.Empty(t, got.RefreshToken, "there is nothing to carry forward")
	assert.Equal(t, "new-access", got.AccessToken)
}

// TestFromRefreshHandlesNilResponse checks a missing response is survivable.
func TestFromRefreshHandlesNilResponse(t *testing.T) {
	t.Parallel()

	got := FromRefresh(nil, "previous")

	assert.Equal(t, "previous", got.RefreshToken)
}

// TestFromOAuth2DropsUnrelatedFields checks the copy is a subset.
func TestFromOAuth2DropsUnrelatedFields(t *testing.T) {
	t.Parallel()

	got := FromOAuth2(&oauth2.Token{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		Expiry:       time.Unix(1800000000, 0).UTC(),
	})

	assert.Equal(t, "access", got.AccessToken)
	assert.Equal(t, "refresh", got.RefreshToken)
	assert.Equal(t, "Bearer", got.TokenType)
	assert.Equal(t, time.Unix(1800000000, 0).UTC(), got.Expiry)
}

// TestFromOAuth2NilIsZero checks a nil response is survivable.
func TestFromOAuth2NilIsZero(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Token{}, FromOAuth2(nil))
}

// TestHasExpiredZeroExpiryNeverExpires covers the never-expiring token.
//
// A zero Expiry means the provider sent no expiry. Treating that as already
// expired would send the user back through an authorization on every request.
func TestHasExpiredZeroExpiryNeverExpires(t *testing.T) {
	t.Parallel()

	assert.False(t, Token{}.HasExpired(time.Now(), 0))
	assert.False(t, Token{}.HasExpired(time.Now(), time.Hour))
}

// TestHasExpiredMarginCoversTheBoundary covers the exact instant.
func TestHasExpiredMarginCoversTheBoundary(t *testing.T) {
	t.Parallel()

	now := time.Now()
	token := Token{Expiry: now.Add(time.Minute)}

	assert.True(t, token.HasExpired(now, time.Minute), "at the margin boundary")
	assert.False(t, token.HasExpired(now, 30*time.Second), "outside the margin")
}

// TestHasExpiredNegativeMarginIsZero treats a bad margin as no margin.
func TestHasExpiredNegativeMarginIsZero(t *testing.T) {
	t.Parallel()

	now := time.Now()
	token := Token{Expiry: now.Add(-time.Second)}

	assert.True(t, token.HasExpired(now, -time.Hour))
}

// TestHasExpiredBeforeNow covers an already-expired token.
func TestHasExpiredBeforeNow(t *testing.T) {
	t.Parallel()

	token := Token{Expiry: time.Now().Add(-time.Minute)}

	assert.True(t, token.HasExpired(time.Now(), 0))
}
