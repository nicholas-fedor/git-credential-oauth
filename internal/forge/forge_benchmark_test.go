// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package forge_test benchmarks forge host detection.
package forge_test

import (
	"testing"

	"github.com/nicholas-fedor/git-credential-oauth/internal/forge"
)

func BenchmarkDetect(b *testing.B) {
	registry := forge.New()
	header := "Basic realm=\"GitHub\"\nBasic realm=\"GitLab\""

	b.ReportAllocs()

	for b.Loop() {
		forge.Detect(registry, "github.com", header)
	}
}
