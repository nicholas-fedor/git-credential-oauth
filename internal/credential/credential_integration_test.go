// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package credential_test exercises the credential wire format through public APIs.
package credential_test

import (
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/nicholas-fedor/git-credential-oauth/internal/credential"
)

// TestPipeGoldenTranscript parses and writes a golden transcript over pipes.
func TestPipeGoldenTranscript(t *testing.T) {
	t.Parallel()

	const input = "url=https://oauth2:token@github.com/nicholas-fedor/git-credential-oauth.git\n" +
		"capability[]=authtype\n" +
		"wwwauth[]=Basic realm=\"GitHub\"\n"

	got := readThroughPipe(t, input)
	same(t, "https", got.Protocol)
	same(t, "github.com", got.Host)
	same(t, "oauth2", got.Username)
	same(t, "token", got.Password)
	same(t, "/nicholas-fedor/git-credential-oauth.git", got.Path)
	same(t, []string{"authtype"}, got.Capability)
	same(t, []string{`Basic realm="GitHub"`}, got.WWWAuth)

	const golden = "capability[]=authtype\n" +
		"authtype=Bearer\n" +
		"credential=access-token\n" +
		"password_expiry_utc=1700000000\n" +
		"oauth_refresh_token=refresh-token\n"

	written := writeThroughPipe(t, goldenResponse())
	same(t, golden, written)
}

// same fails the test when want and got differ.
//
// Parameters:
//   - t: test handle.
//   - want: expected value.
//   - got: actual value.
func same(t *testing.T, want, got any) {
	t.Helper()

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// noError fails the test when err is non-nil.
//
// Parameters:
//   - t: test handle.
//   - err: error to check.
func noError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

// goldenResponse builds the expected helper reply.
//
// Returns:
//   - *credential.Response: golden protocol reply.
func goldenResponse() *credential.Response {
	response := credential.NewResponse()
	response.Add(credential.Capability, "authtype")
	response.Add(credential.Authtype, "Bearer")
	response.Add(credential.Credential, "access-token")
	response.Add(credential.PasswordExpiryUTC, "1700000000")
	response.Add(credential.OAuthRefreshToken, "refresh-token")

	return response
}

// readThroughPipe parses transcript from the read end of a pipe.
//
// Parameters:
//   - t: test handle.
//   - transcript: protocol text written to the pipe.
//
// Returns:
//   - credential.Request: parsed request.
func readThroughPipe(t *testing.T, transcript string) credential.Request {
	t.Helper()

	reader, writer := io.Pipe()
	errc := make(chan error, 1)

	go func() {
		_, err := io.WriteString(writer, transcript)
		errc <- errors.Join(err, writer.Close())
	}()

	request, err := credential.Parse(reader)
	noError(t, err)
	noError(t, <-errc)

	return request
}

// writeThroughPipe writes response to a pipe and returns the transcript.
//
// Parameters:
//   - t: test handle.
//   - response: protocol reply.
//
// Returns:
//   - string: bytes read from the pipe.
func writeThroughPipe(t *testing.T, response *credential.Response) string {
	t.Helper()

	reader, writer := io.Pipe()
	errc := make(chan error, 1)

	go func() {
		err := response.Write(writer)
		errc <- errors.Join(err, writer.Close())
	}()

	payload, err := io.ReadAll(reader)
	noError(t, err)
	noError(t, <-errc)

	return string(payload)
}
