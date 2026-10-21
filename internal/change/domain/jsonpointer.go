// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

const jsonPointerSeparator = "/"

var (
	referenceTokenEscaper   = strings.NewReplacer("~", "~0", "/", "~1")
	referenceTokenUnescaper = strings.NewReplacer("~1", "/", "~0", "~")
)

var ErrMalformedJSONPointer = errors.New("JSON pointer is malformed")

type JSONPointer struct {
	tokens []string
}

func ParseJSONPointer(stated string) (JSONPointer, error) {
	escapedTokens, isRooted := strings.CutPrefix(stated, jsonPointerSeparator)
	if !isRooted {
		return JSONPointer{}, fmt.Errorf("%w: %q does not start at the root", ErrMalformedJSONPointer, stated)
	}
	var tokens []string
	for _, escaped := range strings.Split(escapedTokens, jsonPointerSeparator) {
		token, err := parseReferenceToken(escaped)
		if err != nil {
			return JSONPointer{}, fmt.Errorf("%w: %q", err, stated)
		}
		tokens = append(tokens, token)
	}
	return JSONPointer{tokens: tokens}, nil
}

func (pointer JSONPointer) Tokens() []string { return slices.Clone(pointer.tokens) }

func parseReferenceToken(escaped string) (string, error) {
	token := referenceTokenUnescaper.Replace(escaped)
	if referenceTokenEscaper.Replace(token) != escaped {
		return "", fmt.Errorf("%w: %q is not an escaped reference token", ErrMalformedJSONPointer, escaped)
	}
	return token, nil
}
