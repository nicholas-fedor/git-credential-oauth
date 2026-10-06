// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package callback

import (
	"fmt"
	"html"
)

// projectURL is the repository the success page credits.
//
// The page is served from a loopback listener during an authorization, so the
// link is the only place the user can find where the tool came from.
const projectURL = "https://github.com/nicholas-fedor/" +
	"git-credential-oauth"

// projectName is the short name shown as the link text.
const projectName = "git-credential-oauth"

// successTemplate is the HTML page shown after the provider redirects.
const successTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
	<title>Git authentication</title>
	<meta name="color-scheme" content="light dark" />
</head>
<body>
<p>Success. You may close this page and return to Git.</p>
<p style="font-style: italic">&mdash;<a href="` + projectURL + `">` +
	projectName + `</a> %s</p>
</body>
</html>
`

// successPage renders the callback success page.
//
// The version string is escaped before it is inserted. The page links to the
// git-credential-oauth repository and does not echo the callback query.
//
// Parameters:
//   - version: Application version copied from [Factory.Version].
//
// Returns:
//   - []byte: html document for the success response.
func successPage(version string) []byte {
	return fmt.Appendf(nil, successTemplate, html.EscapeString(version))
}
