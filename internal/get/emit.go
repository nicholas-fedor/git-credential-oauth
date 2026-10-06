// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get

import (
	"slices"
	"strconv"
	"time"

	"github.com/nicholas-fedor/git-credential-oauth/internal/credential"
	"github.com/nicholas-fedor/git-credential-oauth/internal/forge"
	"github.com/nicholas-fedor/git-credential-oauth/internal/oauth"
)

// bearerValue is the authtype emitted when git offered Bearer support.
const bearerValue = "Bearer"

// emit builds an ordered credential response from a token.
//
// capability[] is emitted only when git offered authtype. Bearer is used
// only when that offer, opts.UseBearer, and SupportsBearer all hold.
// password_expiry_utc is expiry minus margin, never before now.
//
// Parameters:
//   - req: git credential request. Capability gates authtype.
//   - detected: host classification. SupportsBearer and DefaultUsername are
//     used.
//   - token: acquired token.
//   - opts: bearer option.
//   - now: clock used to clamp expiry.
//   - margin: duration subtracted from expiry before clamping.
//
// Returns:
//   - credential.Response: ordered credential attributes.
func emit(
	req credential.Request,
	detected forge.Detected,
	token oauth.Token,
	opts Options,
	now time.Time,
	margin time.Duration,
) credential.Response {
	response := credential.NewResponse()
	offered := authtypeOffered(req.Capability)

	if offered {
		response.Add(credential.Capability, credential.Authtype)
	}

	if opts.UseBearer && detected.SupportsBearer() && offered {
		response.Add(credential.Authtype, bearerValue)
		response.Add(credential.Credential, token.AccessToken)
	} else {
		response.Add(credential.Password, token.AccessToken)

		if req.Username == "" {
			response.Add(credential.Username, detected.DefaultUsername())
		}
	}

	if unix, ok := expiryUnix(token.Expiry, now, margin); ok {
		response.Add(credential.PasswordExpiryUTC, strconv.FormatInt(unix, 10))
	}

	if token.RefreshToken != "" {
		response.Add(credential.OAuthRefreshToken, token.RefreshToken)
	}

	return *response
}

// authtypeOffered reports whether git advertised the authtype capability.
//
// Parameters:
//   - capabilities: capability values from the git request.
//
// Returns:
//   - bool: true when authtype was offered.
func authtypeOffered(capabilities []string) bool {
	return slices.Contains(capabilities, credential.Authtype)
}

// expiryUnix returns the clamped password_expiry_utc instant.
//
// A zero expiry omits the key. An instant before now is replaced with now.
//
// Parameters:
//   - expiry: token expiry. Zero omits the key.
//   - now: clock used for the clamp.
//   - margin: duration subtracted before clamping.
//
// Returns:
//   - int64: unix seconds to emit.
//   - bool: true when the key should be emitted.
func expiryUnix(expiry, now time.Time, margin time.Duration) (int64, bool) {
	if expiry.IsZero() {
		return 0, false
	}

	emitAt := expiry.Add(-margin)
	if emitAt.Before(now) {
		emitAt = now
	}

	return emitAt.Unix(), true
}
