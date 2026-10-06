// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package oauth acquires OAuth2 tokens for a configured client.
//
// Three grants are supported: an authorization code with mandatory PKCE S256, a
// device authorization grant for hosts that cannot take a loopback redirect,
// and a refresh. Because this is a public client there is no client secret to
// protect, which is why PKCE is required rather than optional.
//
// The interfaces for the browser, the callback listener, the device prompter,
// and the HTTP transport are declared here, in the consuming package, so a
// caller can substitute any of them.
package oauth
