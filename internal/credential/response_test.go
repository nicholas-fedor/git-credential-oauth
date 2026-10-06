// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package credential

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// TestResponsePreservesAddOrder emits attributes in call order.
func TestResponsePreservesAddOrder(t *testing.T) {
	t.Parallel()

	response := NewResponse()
	response.Add(Capability, "state")
	response.Add(Password, "a=b")
	response.Add(Username, "oauth2")

	const want = "capability[]=state\npassword=a=b\nusername=oauth2\n"
	same(t, want, response.String())

	var builder strings.Builder

	noError(t, response.Write(&builder))
	same(t, response.String(), builder.String())

	pairs := response.Pairs()
	if len(pairs) != 3 {
		t.Fatalf("got %d pairs", len(pairs))
	}

	pairs[0].Key = "mutated"
	same(t, Capability, response.Pairs()[0].Key)
}

// TestNilResponseWrite returns an error.
func TestNilResponseWrite(t *testing.T) {
	t.Parallel()

	var response *Response

	err := response.Write(io.Discard)
	errorIs(t, err, errNilResponse)
}

// TestResponseWriteError wraps the writer failure.
func TestResponseWriteError(t *testing.T) {
	t.Parallel()

	response := NewResponse()
	response.Add(Protocol, httpsProtocol)

	sentinel := errors.New("write failed")
	err := response.Write(errWriter{err: sentinel})
	errorIs(t, err, sentinel)
}

// TestEmptyResponseString emits nothing.
func TestEmptyResponseString(t *testing.T) {
	t.Parallel()

	response := NewResponse()
	if response.String() != "" {
		t.Fatalf("got %q", response.String())
	}

	if len(response.Pairs()) != 0 {
		t.Fatalf("got %#v", response.Pairs())
	}
}

// TestResponseWriteRefusesAnInjectedAttribute keeps one value one attribute.
//
// A token comes from the forge's token endpoint. One containing a newline
// would add attributes of its own, such as a username or an expiry, to what
// Git reads. Nothing may be written in that case, not even the attributes
// before it.
func TestResponseWriteRefusesAnInjectedAttribute(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"token\nusername=attacker",
		"token\n",
		"token\x00tail",
		"token\r",
		"token\r\nusername=attacker",
	} {
		response := NewResponse()
		response.Add(Username, "oauth2")
		response.Add(Password, value)

		var builder strings.Builder

		err := response.Write(&builder)
		errorIs(t, err, ErrUnsafeAttribute)
		same(t, "", builder.String())

		if strings.Contains(err.Error(), "token") {
			t.Fatalf("error %q repeats the secret value", err)
		}
	}
}

// TestResponseWriteRefusesAnUnreadableKey covers keys that cannot round-trip.
func TestResponseWriteRefusesAnUnreadableKey(t *testing.T) {
	t.Parallel()

	for _, key := range []string{"", "a=b", "a\nb", "a\x00b"} {
		response := NewResponse()
		response.Add(key, "value")

		errorIs(t, response.Write(io.Discard), ErrUnsafeAttribute)
	}
}

// TestResponseWriteAllowsEqualsInValues checks what is safe.
//
// Only the first "=" separates key from value, so later ones are part of the
// value and do not add attributes.
func TestResponseWriteAllowsEqualsInValues(t *testing.T) {
	t.Parallel()

	response := NewResponse()
	response.Add(Password, "a=b=c")

	noError(t, response.Write(io.Discard))
}
