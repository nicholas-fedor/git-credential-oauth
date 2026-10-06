// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	json "encoding/json/v2"
)

// Metadata is the subset of RFC 8414 this client uses.
//
// The JSON names are the wire names RFC 8414 defines, so they do not follow Go
// naming.
//
//nolint:tagliatelle // RFC 8414 fixes these names on the wire.
type Metadata struct {
	// Issuer is the identifier of this authorization server.
	Issuer string `json:"issuer"`

	// AuthorizationEndpoint is where the user authorizes.
	AuthorizationEndpoint string `json:"authorization_endpoint"`

	// TokenEndpoint is where a code or refresh token is exchanged.
	TokenEndpoint string `json:"token_endpoint"`

	// DeviceAuthorizationEndpoint is where a device grant starts, when offered.
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`

	// CodeChallengeMethodsSupported lists the PKCE methods this server accepts.
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
}

const (
	// metadataPath is the well-known location of authorization server metadata.
	//
	// RFC 8414 section 3 appends this path to the issuer URL.
	metadataPath = "/.well-known/oauth-authorization-server"

	// metadataLimit bounds a metadata document.
	//
	// A host that answers with an unbounded body must not be able to exhaust the
	// helper's memory during a credential request.
	metadataLimit = 1 << 20

	// metadataTimeout bounds one metadata fetch.
	//
	// Without it a host that accepts a connection and never answers would hang a
	// credential request.
	metadataTimeout = 10 * time.Second
)

var (
	// errMetadataAbsent means the host publishes no metadata document.
	errMetadataAbsent = errors.New("no authorization server metadata")

	// errMetadataStatus means the metadata endpoint answered unexpectedly.
	errMetadataStatus = errors.New("authorization server metadata status")

	// errMetadataBody means the metadata document could not be decoded.
	errMetadataBody = errors.New("authorization server metadata body")

	// errMetadataIssuer means the document named a different host.
	errMetadataIssuer = errors.New("authorization server metadata issuer")
)

// Discover reads authorization server metadata for host.
//
// RFC 9700 section 2.6 recommends that a client use RFC 8414 metadata when it
// is available, and names misconfigured endpoint URLs as the reason: an
// endpoint copied into a wrong host belongs to somebody else. A provider that
// publishes metadata therefore gets its endpoints from the provider rather than
// from this table.
//
// A missing document is not a failure. Most forges do not publish one, so the
// caller falls back to the built-in or derived client.
//
// Only https issuers are fetched. A metadata document can redirect an
// authorization flow anywhere, so a cleartext fetch would be a downgrade.
//
// Parameters:
//   - ctx: cancellation and deadline for the fetch.
//   - host: Git hostname, including a port when present.
//   - client: HTTP client. Nil uses a client with a bounded timeout.
//
// Returns:
//   - metadata: published endpoints and capabilities.
//   - found: false when the host publishes no document.
//   - error: the fetch failed for a reason other than the document being
//     absent.
func Discover(
	ctx context.Context,
	host string,
	client *http.Client,
) (Metadata, bool, error) {
	if client == nil {
		client = &http.Client{Timeout: metadataTimeout}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		issuerHostPath(host),
		http.NoBody,
	)
	if err != nil {
		return Metadata{}, false, fmt.Errorf("build metadata request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return Metadata{}, false, fmt.Errorf("fetch metadata: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	acceptErr := acceptMetadata(resp)
	if acceptErr != nil {
		return Metadata{}, false, acceptErr
	}

	return parseMetadata(resp.Body, host)
}

// SupportsDeviceFlow reports whether the document offers the device grant.
//
// RFC 9700 section 2.5 prefers the grant a provider advertises over one the
// client assumes, so this is how a self-hosted provider that moved its device
// endpoint is discovered.
//
// Returns:
//   - bool: true when the document names a device authorization endpoint.
func (m Metadata) SupportsDeviceFlow() bool {
	return m.DeviceAuthorizationEndpoint != ""
}

// SupportsPKCES256 reports whether the provider accepts an S256 challenge.
//
// RFC 7636 section 4.4 has the client verify this. A provider that lists only
// plain would reject the S256 challenge this client always sends.
//
// Returns:
//   - bool: true when S256 is listed. A document that lists nothing is assumed
//     to accept it, because RFC 6749 does not make the list mandatory.
func (m Metadata) SupportsPKCES256() bool {
	if len(m.CodeChallengeMethodsSupported) == 0 {
		return true
	}

	return slices.Contains(m.CodeChallengeMethodsSupported, "S256")
}

// issuerHostPath returns the metadata URL for a host.
//
// RFC 8414 section 3 places metadata under the issuer, and for a host that is
// itself the issuer the well-known path sits at the root.
//
// Parameters:
//   - host: Git hostname, including a port when present.
//
// Returns:
//   - string: absolute metadata URL.
func issuerHostPath(host string) string {
	return schemeHTTPS + "://" + host + metadataPath
}

// acceptMetadata maps a metadata response status to success or absence.
//
// A host with no metadata answers 404. A host that will not answer at all
// answers 405, which some servers send for an unknown well-known path.
//
// Parameters:
//   - resp: metadata response.
//
// Returns:
//   - error: nil when the document should be read, errMetadataAbsent when the
//     host publishes none.
func acceptMetadata(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound, http.StatusMethodNotAllowed:
		return errMetadataAbsent
	default:
		return fmt.Errorf("%w: %s", errMetadataStatus, resp.Status)
	}
}

// parseMetadata decodes and validates a metadata document.
//
// The issuer has to match the host that was asked. A document naming a
// different issuer is the redirect this feature exists to prevent.
//
// encoding/json/v2 is used rather than the v1 package for two reasons. It
// rejects a document that names the same field twice, and a second
// token_endpoint would make the first one a decoy. It also rejects invalid
// UTF-8 rather than replacing it with U+FFFD, so a mangled issuer cannot be
// silently repaired into a different hostname.
//
// The import is aliased to json because that is the package clause, while the
// path ends in v2.
//
// Parameters:
//   - body: metadata document.
//   - host: Git hostname the document was fetched from.
//
// Returns:
//   - metadata: published endpoints and capabilities.
//   - found: true when the document was usable.
//   - error: the body was unreadable, not JSON, or named another issuer.
func parseMetadata(body io.Reader, host string) (Metadata, bool, error) {
	var meta Metadata

	// The reader is bounded so a host cannot answer with an unbounded body and
	// exhaust this process during a credential request.
	readErr := json.UnmarshalRead(io.LimitReader(body, metadataLimit), &meta)
	if readErr != nil {
		return Metadata{}, false, fmt.Errorf("%w: %w", errMetadataBody, readErr)
	}

	issuerErr := validateIssuer(meta.Issuer, host)
	if issuerErr != nil {
		return Metadata{}, false, issuerErr
	}

	return meta, true, nil
}

// validateIssuer checks the document belongs to the host that served it.
//
// RFC 8414 section 3.3 requires the issuer to match the URL the metadata was
// fetched from. A mismatch means the host served someone else's document, which
// is the exact misconfiguration this feature is meant to catch rather than
// follow.
//
// Parameters:
//   - issuer: issuer field from the document.
//   - host: Git hostname the document was fetched from.
//
// Returns:
//   - error: errMetadataIssuer when the issuer names another host.
func validateIssuer(issuer, host string) error {
	if issuer == "" {
		return fmt.Errorf("%w: document names no issuer", errMetadataIssuer)
	}

	parsed, err := url.Parse(issuer)
	if err != nil {
		return fmt.Errorf("%w: %w", errMetadataIssuer, err)
	}

	if !strings.EqualFold(parsed.Host, host) {
		return fmt.Errorf(
			"%w: %q does not match host %q",
			errMetadataIssuer, issuer, host,
		)
	}

	return nil
}
