// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package forge describes Git hosting forges and their OAuth applications.
//
// A Kind identifies a forge family, such as GitHub Enterprise or Forgejo, and
// a Client carries the registered application for one host. Self-hosted
// families derive their endpoints from the Git remote root, while public hosts
// are looked up in a Registry of built-in clients. Detection reads the
// WWW-Authenticate challenge rather than guessing from the hostname alone.
//
// A built-in client records a host's endpoints and scopes, not an application.
// The operator registers their own on each forge and supplies it through Git
// config. Gitea and Forgejo are the exception, because those servers register
// an application for this helper themselves.
package forge
