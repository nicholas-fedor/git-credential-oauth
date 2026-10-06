// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package credential

import (
	"reflect"
	"testing"
)

// TestEmptyRequestIsTheZeroRequest checks the parser's starting point.
//
// Parsing fills fields in from an empty request, so a default that is not the
// zero value would leak into every request that omits that attribute.
func TestEmptyRequestIsTheZeroRequest(t *testing.T) {
	t.Parallel()

	if got := emptyRequest(); !reflect.DeepEqual(got, Request{}) {
		t.Fatalf("emptyRequest() = %#v, want the zero Request", got)
	}
}
