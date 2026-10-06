// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"fmt"
	"strings"
)

const (
	// realmGitLab is the WWW-Authenticate realm for GitLab.
	realmGitLab = "GitLab"
	// realmGitea is the WWW-Authenticate realm for Gitea.
	realmGitea = "Gitea"
	// realmForgejo is the WWW-Authenticate realm for Forgejo.
	realmForgejo = "Forgejo"
	// realmGitHub is the WWW-Authenticate realm for GitHub.
	realmGitHub = "GitHub"
	// paramRealm is the challenge parameter that carries the realm.
	paramRealm = "realm"
	// paramSepChars separates unquoted challenge parameters.
	paramSepChars = " \t,"
	// tokenTerminators ends an unquoted challenge parameter value.
	tokenTerminators = ",\r\n \t"
)

// ParseChallenge extracts the realm parameter from a WWW-Authenticate header.
//
// Quoted and unquoted values are accepted. Comma-separated auth parameters
// are scanned without treating a comma inside quotes as a separator.
// Challenges separated by newlines are all scanned. A known forge realm
// wins over an earlier unknown realm. A header whose only realm is
// NotGitLab returns NotGitLab; callers must compare the value exactly.
//
// Parameters:
//   - header: raw WWW-Authenticate value, possibly with several challenges.
//
// Returns:
//   - realm: extracted realm value, without surrounding quotes.
//   - ok: true when a realm parameter was present.
func ParseChallenge(header string) (string, bool) {
	if strings.TrimSpace(header) == "" {
		return "", false
	}

	// Prefer a known forge realm when several challenges are present.
	// Otherwise return the first realm, including values such as NotGitLab.
	var first string

	found := false

	for line := range strings.SplitSeq(header, "\n") {
		realm, ok := realmFromChallenge(line)
		if !ok {
			continue
		}

		if knownRealm(realm) {
			return realm, true
		}

		if !found {
			first = realm
			found = true
		}
	}

	if !found {
		return "", false
	}

	return first, true
}

// knownRealm reports whether realm is a forge name Detect understands.
//
// Matching is exact. realm="NotGitLab" is not GitLab.
//
// Parameters:
//   - realm: realm parameter from one challenge.
//
// Returns:
//   - bool: true for GitLab, Gitea, Forgejo, and GitHub.
func knownRealm(realm string) bool {
	switch realm {
	case realmGitLab, realmGitea, realmForgejo, realmGitHub:
		return true
	default:
		return false
	}
}

// realmFromChallenge returns the realm on one challenge line.
//
// The scheme token before the first space is skipped.
//
// Parameters:
//   - challenge: one WWW-Authenticate challenge line.
//
// Returns:
//   - string: realm value when present.
//   - bool: true when a realm parameter was found.
func realmFromChallenge(challenge string) (string, bool) {
	challenge = strings.TrimSpace(challenge)
	if challenge == "" {
		return "", false
	}

	// A scheme token precedes auth-params. A bare realm= form has no scheme.
	params := challenge
	if _, afterScheme, found := strings.Cut(challenge, " "); found {
		params = strings.TrimSpace(afterScheme)
	}

	return scanRealm(params)
}

// scanRealm walks comma-separated auth-params for the first realm.
//
// A parser that does not consume input stops rather than looping.
//
// Parameters:
//   - params: auth-param text after the challenge scheme.
//
// Returns:
//   - string: first realm value.
//   - bool: true when a realm parameter was found.
func scanRealm(params string) (string, bool) {
	rest := params
	for rest != "" {
		rest = strings.TrimLeft(rest, paramSepChars)
		if rest == "" {
			break
		}

		name, afterName, nameOK := cutParamName(rest)
		if !nameOK {
			break
		}

		value, afterValue, valueOK := cutParamValue(afterName)
		if !valueOK {
			break
		}

		if strings.EqualFold(name, paramRealm) {
			return value, true
		}

		// A parser that does not consume input must stop.
		if afterValue == rest {
			break
		}

		rest = afterValue
	}

	return "", false
}

// cutParamName splits an auth-param name from the text after '='.
//
// The name must be non-empty and must be followed by '='.
//
// Parameters:
//   - input: remaining auth-param text.
//
// Returns:
//   - name: parameter name.
//   - rest: text after '='.
//   - ok: true when a name and '=' were found.
//
//nolint:nonamedreturns // Same-type returns need names.
func cutParamName(input string) (name, rest string, ok bool) {
	end := 0
	for end < len(input) {
		switch input[end] {
		case '=', ' ', '\t', ',':
			if end == 0 {
				return "", "", false
			}

			name = input[:end]
			rest = strings.TrimLeft(input[end:], " \t")

			if rest == "" || rest[0] != '=' {
				return "", "", false
			}

			return name, rest[1:], true
		default:
			end++
		}
	}

	return "", "", false
}

// cutParamValue reads a quoted or token auth-param value.
//
// A leading quote selects the quoted-string reader. Otherwise the value ends at
// the first token terminator, which includes a bare carriage return or line
// feed, so an unquoted parameter cannot carry one out either.
//
// Parameters:
//   - input: text after '='.
//
// Returns:
//   - value: decoded parameter value.
//   - rest: unconsumed text.
//   - ok: true when a value was read, including an empty value.
//
//nolint:nonamedreturns // Same-type returns need names.
func cutParamValue(input string) (value, rest string, ok bool) {
	input = strings.TrimLeft(input, " \t")
	if input == "" {
		return "", "", true
	}

	if input[0] == '"' {
		return cutQuoted(input[1:])
	}

	end := 0
	for end < len(input) &&
		!strings.ContainsRune(tokenTerminators, rune(input[end])) {
		end++
	}

	return input[:end], input[end:], true
}

// forbiddenRealmByte reports whether char may not appear inside a realm.
//
// A carriage return or line feed is rejected. A realm carrying either would
// let a header value forge a second transcript line, so a challenge holding
// one has no usable realm.
//
// Parameters:
//   - char: byte read from a quoted string, escaped or literal.
//
// Returns:
//   - bool: true for a carriage return or line feed.
func forbiddenRealmByte(char byte) bool {
	return char == '\r' || char == '\n'
}

// cutQuoted reads a quoted-string, honoring backslash escapes.
//
// The opening quote has already been consumed. An unclosed quote fails. A
// carriage return or line feed fails, escaped or not.
//
// Parameters:
//   - input: text inside and after the opening quote.
//
// Returns:
//   - value: decoded quoted string.
//   - rest: text after the closing quote.
//   - ok: true when a closing quote was found and no forbidden byte was read.
//
//nolint:nonamedreturns // Same-type returns need names.
func cutQuoted(input string) (value, rest string, ok bool) {
	var builder strings.Builder

	for index := 0; index < len(input); index++ {
		char := input[index]
		if char == '\\' {
			if index+1 >= len(input) {
				return "", "", false
			}

			escaped := input[index+1]
			if forbiddenRealmByte(escaped) {
				return "", "", false
			}

			err := writeQuotedByte(&builder, escaped)
			if err != nil {
				return "", "", false
			}

			index++

			continue
		}

		if char == '"' {
			return builder.String(), input[index+1:], true
		}

		if forbiddenRealmByte(char) {
			return "", "", false
		}

		err := writeQuotedByte(&builder, char)
		if err != nil {
			return "", "", false
		}
	}

	return "", "", false
}

// writeQuotedByte appends char to builder.
//
// [strings.Builder.WriteByte] is documented to return nil. The error is still
// checked and returned.
//
// Parameters:
//   - builder: destination for the decoded byte.
//   - char: byte to append.
//
// Returns:
//   - error: non-nil when the write fails.
func writeQuotedByte(builder *strings.Builder, char byte) error {
	err := builder.WriteByte(char)
	if err != nil {
		return fmt.Errorf("write quoted byte: %w", err)
	}

	return nil
}
