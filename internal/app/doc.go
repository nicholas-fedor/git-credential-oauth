// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package app is the git-credential-oauth composition root.
//
// It is the only package that names the process streams, looks up the git
// binary, or constructs the logger. Commands return errors. This package maps
// them to process status codes.
package app
