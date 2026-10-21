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
		for i := 0; i < len(escaped); i++ {
			if escaped[i] == '~' && (i+1 == len(escaped) || (escaped[i+1] != '0' && escaped[i+1] != '1')) {
				return JSONPointer{}, fmt.Errorf("%w: %q", ErrMalformedJSONPointer, stated)
			}
		}
		tokens = append(tokens, unescapedToken(escaped))
	}
	return JSONPointer{tokens: tokens}, nil
}

func (pointer JSONPointer) Tokens() []string { return pointer.tokens }

// RFC 6901 §4: "~1" is decoded before "~0", so that "~01" yields "~1" and not "/".
func unescapedToken(escaped string) string {
	return strings.ReplaceAll(strings.ReplaceAll(escaped, "~1", "/"), "~0", "~")
}
