// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
)

var ErrMalformedJSONPointer = errors.New("JSON pointer is malformed")

type JSONPointer struct {
	tokens []string
}

func ParseJSONPointer(stated string) (JSONPointer, error) {
	if !strings.HasPrefix(stated, "/") {
		return JSONPointer{}, fmt.Errorf("%w: %q", ErrMalformedJSONPointer, stated)
	}
	var tokens []string
	for _, escaped := range strings.Split(stated, "/")[1:] {
		token, err := parseReferenceToken(escaped)
		if err != nil {
			return JSONPointer{}, fmt.Errorf("%w: %q", err, stated)
		}
		tokens = append(tokens, token)
	}
	return JSONPointer{tokens: tokens}, nil
}

func (pointer JSONPointer) Tokens() []string { return pointer.tokens }

func parseReferenceToken(escaped string) (string, error) {
	for at, char := range escaped {
		if char == '~' && !startsWithEscapeCode(escaped[at+1:]) {
			return "", ErrMalformedJSONPointer
		}
	}
	return strings.ReplaceAll(strings.ReplaceAll(escaped, "~1", "/"), "~0", "~"), nil
}

func startsWithEscapeCode(rest string) bool {
	return strings.HasPrefix(rest, "0") || strings.HasPrefix(rest, "1")
}
