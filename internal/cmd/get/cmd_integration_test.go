// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package get_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/get"
	mockGet "github.com/nicholas-fedor/git-credential-oauth/internal/cmd/get/mocks"
	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd/options"
	"github.com/nicholas-fedor/git-credential-oauth/internal/credential"
	getsvc "github.com/nicholas-fedor/git-credential-oauth/internal/get"
)

// request is the transcript git sends for a public HTTPS host.
const request = "protocol=https\nhost=github.com\n\n"

// failReader fails every read so the parse error path is reachable.
type failReader struct{}

func (failReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

// failWriter fails every write so the response error path is reachable.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("write output: closed pipe")
}

// TestNewCommand checks the metadata the git credential protocol requires.
//
// git identifies operations by name and passes no arguments, so Use must be
// exactly "get" and positional arguments must be rejected.
func TestNewCommand(t *testing.T) {
	t.Parallel()

	command := get.NewCommand(
		t.Context(),
		strings.NewReader(request),
		io.Discard,
		mockGet.NewMockGetter(t),
		options.New(),
	)

	assert.Equal(t, "get", command.Name())
	require.NoError(t, command.Args(command, nil))
	require.Error(t, command.Args(command, []string{"extra"}))
}

// TestRunWritesTheResolvedResponse covers the protocol transcript.
//
// The response is the only thing git reads, so it must reach the writer
// unchanged.
func TestRunWritesTheResolvedResponse(t *testing.T) {
	t.Parallel()

	resp := credential.NewResponse()
	resp.Add(credential.Username, "git-credential-oauth")
	resp.Add(credential.Password, "token-value")

	getter := mockGet.NewMockGetter(t)
	getter.EXPECT().
		Get(mock.Anything, mock.Anything, flow(false, false)).
		Return(*resp, nil).
		Once()

	var stdout bytes.Buffer

	err := execute(t, strings.NewReader(request), &stdout, getter, options.New())
	require.NoError(t, err)

	assert.Equal(t, resp.String(), stdout.String())
}

// TestRunPassesTheParsedRequest checks the request reaching the service.
//
// The forge is chosen from the requested host, so a request that arrived
// unparsed would authenticate against the wrong host.
func TestRunPassesTheParsedRequest(t *testing.T) {
	t.Parallel()

	getter := mockGet.NewMockGetter(t)
	getter.EXPECT().
		Get(mock.Anything, credential.Request{
			Protocol: "https",
			Host:     "github.com",
		}, flow(false, false)).
		Return(*credential.NewResponse(), nil).
		Once()

	err := execute(t, strings.NewReader(request), io.Discard, getter, options.New())
	require.NoError(t, err)
}

// TestRunForwardsFlowFlags checks the persistent flags reach the service.
//
// The flow is chosen when the helper is configured, so a get that ignored the
// flag would silently fall back to the browser flow.
func TestRunForwardsFlowFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		opts   *options.Options
		name   string
		bearer bool
		device bool
	}{
		{name: "defaults", opts: options.New()},
		{name: "device flow", opts: &options.Options{Device: true}, device: true},
		{name: "bearer preference", opts: &options.Options{Bearer: true}, bearer: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			getter := mockGet.NewMockGetter(t)
			getter.EXPECT().
				Get(mock.Anything, mock.Anything, flow(tt.bearer, tt.device)).
				Return(*credential.NewResponse(), nil).
				Once()

			err := execute(t, strings.NewReader(request), io.Discard, getter, tt.opts)
			require.NoError(t, err)
		})
	}
}

// TestRunFailures checks that each stage stops the command and keeps the
// cause reachable through errors.Is.
func TestRunFailures(t *testing.T) {
	t.Parallel()

	t.Run("an unreadable request never reaches the getter", func(t *testing.T) {
		t.Parallel()

		getter := mockGet.NewMockGetter(t)

		err := execute(t, failReader{}, io.Discard, getter, options.New())
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)

		getter.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("a getter failure is not swallowed", func(t *testing.T) {
		t.Parallel()

		boom := errors.New("no stored credential")

		getter := mockGet.NewMockGetter(t)
		getter.EXPECT().
			Get(mock.Anything, mock.Anything, mock.Anything).
			Return(credential.Response{}, boom).
			Once()

		err := execute(t, strings.NewReader(request), io.Discard, getter, options.New())
		require.ErrorIs(t, err, boom)
	})

	t.Run("an unwritable response is reported", func(t *testing.T) {
		t.Parallel()

		getter := mockGet.NewMockGetter(t)
		getter.EXPECT().
			Get(mock.Anything, mock.Anything, mock.Anything).
			Return(*credential.NewResponse(), nil).
			Once()

		err := execute(t, strings.NewReader(request), failWriter{}, getter, options.New())
		require.Error(t, err)
	})
}

// execute runs the get command with no positional arguments.
func execute(
	t *testing.T,
	stdin io.Reader,
	stdout io.Writer,
	getter get.Getter,
	opts *options.Options,
) error {
	t.Helper()

	command := get.NewCommand(t.Context(), stdin, stdout, getter, opts)
	command.SetArgs([]string{})

	return command.ExecuteContext(t.Context())
}

// flow matches the options the command builds from its persistent flags.
func flow(bearer, device bool) any {
	return mock.MatchedBy(func(opts getsvc.Options) bool {
		return opts.UseBearer == bearer && opts.Device == device
	})
}

// compile-time check that the generated mock satisfies the interface.
var _ get.Getter = (*mockGet.MockGetter)(nil)
