// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package browser

// Operating system names that select a browser command.
const (
	// goosDarwin selects the macOS open command.
	goosDarwin = "darwin"

	// goosWindows selects the Windows start command.
	goosWindows = "windows"
)

// Command returns the platform command that opens targetURL.
//
// The command is the same executable a user would run by hand, and the URL is
// passed as its final argument. An unrecognized goos uses xdg-open, which is
// correct on the remaining platforms and is also the Linux default.
//
// Parameters:
//   - goos: operating system name, such as linux, darwin, or windows.
//   - targetURL: URL to open.
//
// Returns:
//   - string: command executable name.
//   - []string: command arguments, ending with targetURL.
func Command(goos, targetURL string) (string, []string) {
	switch goos {
	case goosDarwin:
		return "open", []string{targetURL}
	case goosWindows:
		// start is a shell built-in, so cmd.exe carries it.
		return "cmd", []string{"/c", "start", "", targetURL}
	default:
		return "xdg-open", []string{targetURL}
	}
}
