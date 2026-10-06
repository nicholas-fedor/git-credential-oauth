// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"reflect"
	"testing"
)

// equal fails the test when want and got differ.
//
// Parameters:
//   - t: test handle.
//   - want: expected value.
//   - got: actual value.
func equal(t *testing.T, want, got any) {
	t.Helper()

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestParseChallenge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
		want   string
		ok     bool
	}{
		{name: "empty", header: "", ok: false},
		{name: "whitespace", header: " \t\n", ok: false},
		{name: "quoted gitlab", header: `Basic realm="GitLab"`, want: realmGitLab, ok: true},
		{name: "not gitlab", header: `Basic realm="NotGitLab"`, want: "NotGitLab", ok: true},
		{name: "unquoted", header: "Basic realm=GitLab", want: realmGitLab, ok: true},
		{
			name:   "comma separated",
			header: `Basic realm="GitLab", charset="UTF-8"`,
			want:   realmGitLab,
			ok:     true,
		},
		{
			name:   "spaces around equals",
			header: `Basic realm = "Gitea"`,
			want:   realmGitea,
			ok:     true,
		},
		{
			name:   "later challenge",
			header: "Basic charset=\"UTF-8\"\nBearer realm=\"Forgejo\"",
			want:   realmForgejo,
			ok:     true,
		},
		{
			name:   "known realm wins over earlier unknown",
			header: "Basic realm=\"NotGitLab\"\nBasic realm=\"GitLab\"",
			want:   realmGitLab,
			ok:     true,
		},
		{
			name:   "quoted comma",
			header: `Basic realm="Git,Lab"`,
			want:   "Git,Lab",
			ok:     true,
		},
		{
			name:   "escaped quote",
			header: `Basic realm="a\"b"`,
			want:   `a"b`,
			ok:     true,
		},
		{name: "unterminated", header: `Basic realm="GitLab`, ok: false},
		{name: "scheme only", header: "Basic", ok: false},
		{name: "case insensitive name", header: `Basic REALM="GitHub"`, want: realmGitHub, ok: true},
		{name: "carriage return", header: "Basic realm=\"Gitea\"\r\n", want: realmGitea, ok: true},
		{
			name:   "unquoted carriage return ends token",
			header: "Basic realm=GitLab\rX",
			want:   realmGitLab,
			ok:     true,
		},
		{
			name:   "unquoted carriage return yields empty realm",
			header: "reAlm=\r0",
			ok:     true,
		},
		{
			name:   "quoted carriage return rejected",
			header: "Basic realm=\"Git\rLab\"",
			ok:     false,
		},
		{
			name:   "escaped carriage return rejected",
			header: "Basic realm=\"Git\\\rLab\"",
			ok:     false,
		},
		{
			name:   "poisoned challenge does not hide a later realm",
			header: "Basic realm=\"Bad\rRealm\"\nBearer realm=\"Gitea\"",
			want:   realmGitea,
			ok:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ParseChallenge(tt.header)
			equal(t, tt.ok, ok)
			equal(t, tt.want, got)
		})
	}
}
