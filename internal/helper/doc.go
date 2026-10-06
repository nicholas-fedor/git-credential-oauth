// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package helper installs and removes the git-credential-oauth helper.
//
// It owns the credential.helper configuration value. Configure resets the
// helper list, appends a storage helper, and appends this program last, so
// another helper can supply an existing credential before an interactive
// authorization is needed. Unconfigure is the exact inverse.
package helper
