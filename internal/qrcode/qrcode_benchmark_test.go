// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package qrcode_test

import (
	"io"
	"testing"

	"github.com/nicholas-fedor/git-credential-oauth/internal/qrcode"
)

func BenchmarkRender(b *testing.B) {
	for b.Loop() {
		err := qrcode.Render(io.Discard, "https://example.com/device")
		if err != nil {
			b.Fatal(err)
		}
	}
}
