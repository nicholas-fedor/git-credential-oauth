// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package callback

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSuccessPageEscapesTheVersion checks build metadata cannot inject markup.
//
// The version is stamped at link time and is not trusted input, but it is
// printed into HTML, so it is escaped like any other value.
func TestSuccessPageEscapesTheVersion(t *testing.T) {
	t.Parallel()

	page := string(successPage(`<script>alert("x")</script>`))

	assert.NotContains(t, page, "<script>")
	assert.Contains(t, page, "&lt;script&gt;")
}

// TestSuccessPageTellsTheUserToReturnToGit checks the page's one job.
func TestSuccessPageTellsTheUserToReturnToGit(t *testing.T) {
	t.Parallel()

	page := string(successPage("v1.2.3"))

	assert.True(t, strings.HasPrefix(page, "<!DOCTYPE html>"))
	assert.Contains(t, page, "return to Git")
	assert.Contains(t, page, `href="`+projectURL+`"`)
	assert.Contains(t, page, "v1.2.3")
	assert.NotContains(t, page, "%!", "every format verb was filled")
}
