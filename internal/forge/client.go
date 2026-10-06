// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"golang.org/x/oauth2"
)

// Client is the OAuth application registered for one Git host.
type Client struct {
	Host         string
	ClientID     string
	ClientSecret string
	Endpoint     oauth2.Endpoint
	Scopes       []string
	Kind         Kind
	PKCE         bool
	DeviceFlow   bool
}

const (
	// usePKCE is set on every derived and built-in client. The authorization
	// code exchange always sends an S256 challenge.
	usePKCE = true

	// pathOAuthAuthorize is the Gitea and Forgejo authorization path.
	pathOAuthAuthorize = "/login/oauth/authorize"
	// pathOAuthAccessToken is the Gitea and Forgejo token path.
	pathOAuthAccessToken = "/login/oauth/access_token"
	// pathGitHubDevice is the GitHub Enterprise device-authorization path.
	pathGitHubDevice = "/login/device/code"
	// pathGitLabAuthorize is the GitLab authorization path.
	pathGitLabAuthorize = "/oauth/authorize"
	// pathGitLabToken is the GitLab token path.
	pathGitLabToken = "/oauth/token"
	// pathGitLabDevice is the GitLab device-authorization path.
	pathGitLabDevice = "/oauth/authorize_device"
	// universalGiteaClientID is the application Gitea and Forgejo register for
	// this helper themselves.
	//
	// It is the one client ID this program carries. Both servers create the
	// application at startup under the name git-credential-oauth, with the
	// loopback redirect, unless an administrator removes it from the instance's
	// default applications. It belongs to the instance rather than to anyone's
	// account, so using it presents this helper as nothing but itself.
	universalGiteaClientID = "a4792ccc-144e-407e-86c9-5e7d8d9c3269"

	// schemeHTTPS is the only transport a remote may use. A cleartext scheme
	// would carry the token in the open, which RFC 9700 section 2.6 forbids.
	schemeHTTPS = "https"
)

var (
	// errNotSelfHosted is returned when endpoints cannot be derived for a kind.
	errNotSelfHosted = errors.New("kind is not self-hosted")
	// errMalformedGitURL is returned when a git URL cannot be parsed.
	errMalformedGitURL = errors.New("git URL is malformed")
	// errMalformedURL is returned when a host-replacement URL cannot be parsed.
	errMalformedURL = errors.New("url is malformed")
	// errParseGitURL wraps a git URL parse failure.
	errParseGitURL = errors.New("parse git URL")
	// errHostEmpty is returned when a git URL has no host.
	errHostEmpty = errors.New("host is empty")
	// errInsecureGitURL is returned when a remote is not https.
	errInsecureGitURL = errors.New("git URL must use https")
)

// Derive builds an OAuth client for a self-hosted forge rooted at gitURL.
//
// The family decides the shape: a Gitea-layout family gets the Gitea paths,
// colon scopes, and the application its server registers, while GitLab and
// GitHub Enterprise get their own paths and no client ID. Non-self-hosted kinds
// return an error. A malformed gitURL returns an error and does not exit the
// process. Callers invoke this at lookup time.
//
// Endpoints are the ones this family is known to use. A provider that
// publishes RFC 8414 metadata may move them, so callers that can make a request
// should run [Discover] and prefer the published endpoints.
//
// Parameters:
//   - kind: forge family. Only self-hosted kinds succeed.
//   - gitURL: absolute Git remote root, such as https://gitlab.example.com.
//
// Returns:
//   - client: derived application, with Host set from gitURL.
//   - err: non-nil when kind is not self-hosted or gitURL is malformed.
func Derive(kind Kind, gitURL string) (Client, error) {
	if !kind.IsSelfHosted() {
		return Client{}, fmt.Errorf("kind %s: %w", kind, errNotSelfHosted)
	}

	host, err := hostFromGitURL(gitURL)
	if err != nil {
		return Client{}, err
	}

	if kind.UsesGiteaLayout() {
		return deriveGiteaLayout(kind, host, gitURL)
	}

	return deriveVendorLayout(kind, host, gitURL)
}

// FromMetadata builds a client from published authorization server metadata.
//
// RFC 9700 section 2.6 recommends using this metadata when a provider publishes
// it, and names misconfigured endpoint URLs as the reason: an endpoint recorded
// for the wrong host belongs to somebody else. Taking the endpoints from the
// provider removes that class of mistake.
//
// A document that names no authorization or token endpoint is not usable, so
// the caller keeps whatever it derived instead.
//
// Parameters:
//   - kind: forge family.
//   - host: Git hostname, including a port when present.
//   - meta: published metadata.
//
// Returns:
//   - client: client using the published endpoints.
//   - found: false when the document was not usable.
//   - error: non-nil when a published endpoint is not https.
func FromMetadata(kind Kind, host string, meta Metadata) (Client, bool, error) {
	if meta.AuthorizationEndpoint == "" || meta.TokenEndpoint == "" {
		return Client{}, false, nil
	}

	endpoint, err := httpsEndpoint(kind, meta)
	if err != nil {
		return Client{}, false, err
	}

	client := newClient(
		host,
		kind,
		publicClientIDFor(kind),
		metadataScopes(kind),
		endpoint,
		meta.SupportsDeviceFlow(),
	)

	client.PKCE = meta.SupportsPKCES256()

	return client, true, nil
}

// httpsEndpoint converts a metadata document into an endpoint.
//
// Only https endpoints are accepted. The authorization response carries the
// token, so a cleartext endpoint would send it in the open, which RFC 9700
// section 2.6 forbids.
//
// Parameters:
//   - kind: forge family, which decides the client authentication style.
//   - meta: published metadata.
//
// Returns:
//   - oauth2.Endpoint: validated endpoint.
//   - error: non-nil when a published endpoint is not https.
func httpsEndpoint(
	kind Kind,
	meta Metadata,
) (oauth2.Endpoint, error) {
	for _, endpoint := range []string{
		meta.AuthorizationEndpoint,
		meta.TokenEndpoint,
		meta.DeviceAuthorizationEndpoint,
	} {
		if endpoint == "" {
			continue
		}

		parsed, err := url.Parse(endpoint)
		if err != nil {
			return oauth2.Endpoint{}, fmt.Errorf("%w: %w", errInsecureGitURL, err)
		}

		if parsed.Scheme != schemeHTTPS {
			return oauth2.Endpoint{}, fmt.Errorf(
				"%w: %q uses %q",
				errInsecureGitURL,
				endpoint,
				parsed.Scheme,
			)
		}
	}

	return oauth2.Endpoint{
		AuthURL:       meta.AuthorizationEndpoint,
		TokenURL:      meta.TokenEndpoint,
		DeviceAuthURL: meta.DeviceAuthorizationEndpoint,
		AuthStyle:     authStyleFor(kind),
	}, nil
}

// authStyleFor returns the client authentication style a family needs.
//
// Parameters:
//   - kind: forge family.
//
// Returns:
//   - oauth2.AuthStyle: in-params for a Gitea-layout family, auto-detect
//     otherwise.
func authStyleFor(kind Kind) oauth2.AuthStyle {
	if kind.UsesGiteaLayout() {
		return oauth2.AuthStyleInParams
	}

	return oauth2.AuthStyleAutoDetect
}

// publicClientIDFor returns the family client ID, empty when one must be
// registered by the operator.
//
// Parameters:
//   - kind: forge family.
//
// Returns:
//   - string: server-registered client ID, or empty for every other family.
func publicClientIDFor(kind Kind) string {
	if kind.UsesPublicClientID() {
		return universalGiteaClientID
	}

	return ""
}

// metadataScopes returns the scopes a family requests.
//
// Parameters:
//   - kind: forge family.
//
// Returns:
//   - []string: scopes for the family, nil where it takes none.
func metadataScopes(kind Kind) []string {
	switch kind {
	case KindForgejo:
		return forgejoScopes()
	case KindGitHubEnterprise:
		return githubScopes()
	case KindGitLab:
		return gitLabScopes()
	default:
		return nil
	}
}

// deriveGiteaLayout builds a Gitea or Forgejo client.
//
// Both families share the Gitea endpoint paths and the application their
// servers register for this helper, so a self-hosted instance works before an
// operator registers anything. Forgejo asks for colon-separated scopes; plain
// Gitea takes none. Neither supports the device grant.
//
// Parameters:
//   - kind: Gitea or Forgejo.
//   - host: Git host, including a port when present.
//   - gitURL: absolute Git remote root.
//
// Returns:
//   - client: derived application.
//   - error: gitURL cannot be parsed.
func deriveGiteaLayout(kind Kind, host, gitURL string) (Client, error) {
	endpoint, err := giteaStyleEndpoint(gitURL)
	if err != nil {
		return Client{}, err
	}

	var scopes []string

	if kind == KindForgejo {
		scopes = forgejoScopes()
	}

	return newClient(
		host,
		kind,
		universalGiteaClientID,
		scopes,
		endpoint,
		kind.SupportsDeviceFlow(),
	), nil
}

// deriveVendorLayout builds a GitLab or GitHub Enterprise client.
//
// Neither server registers an application for this helper, so an operator has
// to register one and set the credential keys. Both support the device grant.
//
// Parameters:
//   - kind: GitLab or GitHub Enterprise.
//   - host: Git host, including a port when present.
//   - gitURL: absolute Git remote root.
//
// Returns:
//   - client: derived application with no client ID.
//   - error: gitURL cannot be parsed.
func deriveVendorLayout(kind Kind, host, gitURL string) (Client, error) {
	var (
		endpoint oauth2.Endpoint
		scopes   []string
		err      error
	)

	if kind == KindGitHubEnterprise {
		endpoint, err = endpointFor(
			gitURL,
			pathOAuthAuthorize,
			pathOAuthAccessToken,
			pathGitHubDevice,
			oauth2.AuthStyleAutoDetect,
		)
		scopes = githubScopes()
	} else {
		endpoint, err = endpointFor(
			gitURL,
			pathGitLabAuthorize,
			pathGitLabToken,
			pathGitLabDevice,
			oauth2.AuthStyleAutoDetect,
		)
		scopes = gitLabScopes()
	}

	if err != nil {
		return Client{}, err
	}

	return newClient(
		host, kind, "", scopes, endpoint, kind.SupportsDeviceFlow(),
	), nil
}

// cloneClient returns a client whose scope slice is independent of client.
//
// The returned value is a shallow copy except for Scopes, which is cloned.
//
// Parameters:
//   - client: registry client to copy.
//
// Returns:
//   - Client: copy that does not share the scope slice.
func cloneClient(client Client) Client {
	client.Scopes = slices.Clone(client.Scopes)

	return client
}

// endpointFor joins gitURL with the authorization, token, and device paths.
//
// An empty devicePath leaves DeviceAuthURL empty.
//
// Parameters:
//   - gitURL: absolute Git remote root.
//   - authPath: authorization endpoint path.
//   - tokenPath: token endpoint path.
//   - devicePath: device authorization path. Empty disables the device URL.
//   - style: OAuth client authentication style.
//
// Returns:
//   - oauth2.Endpoint: joined endpoint.
//   - error: gitURL cannot be parsed.
func endpointFor(
	gitURL string,
	authPath string,
	tokenPath string,
	devicePath string,
	style oauth2.AuthStyle,
) (oauth2.Endpoint, error) {
	authURL, err := appendPath(gitURL, authPath)
	if err != nil {
		return oauth2.Endpoint{}, err
	}

	tokenURL, err := appendPath(gitURL, tokenPath)
	if err != nil {
		return oauth2.Endpoint{}, err
	}

	deviceURL, err := deviceAuthURL(gitURL, devicePath)
	if err != nil {
		return oauth2.Endpoint{}, err
	}

	return oauth2.Endpoint{
		AuthURL:       authURL,
		DeviceAuthURL: deviceURL,
		TokenURL:      tokenURL,
		AuthStyle:     style,
	}, nil
}

// deviceAuthURL joins a device path, or returns empty when path is empty.
//
// Parameters:
//   - gitURL: absolute Git remote root.
//   - devicePath: device authorization path. Empty skips joining.
//
// Returns:
//   - string: joined device URL, or empty when devicePath is empty.
//   - error: gitURL cannot be parsed.
func deviceAuthURL(gitURL, devicePath string) (string, error) {
	if devicePath == "" {
		return "", nil
	}

	deviceURL, err := appendPath(gitURL, devicePath)
	if err != nil {
		return "", err
	}

	return deviceURL, nil
}

// giteaStyleEndpoint builds Gitea and Forgejo authorize and token URLs.
//
// Device flow is unsupported, so DeviceAuthURL stays empty. Client credentials
// are sent in the POST body.
//
// Parameters:
//   - gitURL: absolute Git remote root.
//
// Returns:
//   - oauth2.Endpoint: authorize and token URLs with AuthStyleInParams.
//   - error: gitURL cannot be parsed.
func giteaStyleEndpoint(gitURL string) (oauth2.Endpoint, error) {
	return endpointFor(
		gitURL,
		pathOAuthAuthorize,
		pathOAuthAccessToken,
		"",
		oauth2.AuthStyleInParams,
	)
}

// appendPath validates gitURL and concatenates path.
//
// A trailing slash on gitURL is removed before path is joined.
//
// Parameters:
//   - gitURL: absolute Git remote root.
//   - path: endpoint path, including the leading slash.
//
// Returns:
//   - string: gitURL joined with path.
//   - error: gitURL cannot be parsed.
func appendPath(gitURL, path string) (string, error) {
	_, err := parseGitURL(gitURL)
	if err != nil {
		return "", fmt.Errorf("append %s: %w", path, err)
	}

	return strings.TrimRight(gitURL, "/") + path, nil
}

// hostFromGitURL returns the host of an absolute gitURL.
//
// The host includes a port when one is present.
//
// Parameters:
//   - gitURL: absolute Git remote root.
//
// Returns:
//   - string: URL host.
//   - error: gitURL is not absolute.
func hostFromGitURL(gitURL string) (string, error) {
	parsed, err := parseGitURL(gitURL)
	if err != nil {
		return "", err
	}

	return parsed.Host, nil
}

// parseGitURL parses gitURL and rejects values that are not absolute.
//
// Both scheme and host are required. A cleartext http scheme is rejected too:
// RFC 9700 section 2.6 forbids transmitting an authorization response over an
// unencrypted connection, and a remote of http:// would send the token in the
// clear. RFC 8252 section 7.3 permits http only for the loopback redirect,
// which is built locally and never comes through here.
//
// Parameters:
//   - gitURL: Git remote root to parse.
//
// Returns:
//   - *[url.URL]: parsed absolute https URL.
//   - error: parse failure, a missing scheme or host, or a non-https scheme.
func parseGitURL(gitURL string) (*url.URL, error) {
	parsed, err := url.Parse(gitURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errParseGitURL, err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: %q", errMalformedGitURL, gitURL)
	}

	if parsed.Scheme != schemeHTTPS {
		return nil, fmt.Errorf(
			"%w: %q uses %q",
			errInsecureGitURL, gitURL, parsed.Scheme,
		)
	}

	return parsed, nil
}

// newClient fills every Client field so callers cannot forget one.
//
// PKCE is always enabled. The client secret is always empty: this program
// carries no application secret, so one can only arrive from the operator's
// Git config.
//
// Parameters:
//   - host: Git host, including a port when present.
//   - kind: forge family.
//   - clientID: OAuth client ID, empty when the operator has to supply one.
//   - scopes: OAuth scopes.
//   - endpoint: authorization and token URLs.
//   - deviceFlow: whether the device grant is supported.
//
// Returns:
//   - Client: filled client with PKCE enabled and no secret.
func newClient(
	host string,
	kind Kind,
	clientID string,
	scopes []string,
	endpoint oauth2.Endpoint,
	deviceFlow bool,
) Client {
	return Client{
		Host:         host,
		Kind:         kind,
		ClientID:     clientID,
		ClientSecret: "",
		Scopes:       slices.Clone(scopes),
		Endpoint:     endpoint,
		PKCE:         usePKCE,
		DeviceFlow:   deviceFlow,
	}
}
