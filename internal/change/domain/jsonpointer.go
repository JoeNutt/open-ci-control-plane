// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "strings"

type JSONPointer struct {
	tokens []string
}

func ParseJSONPointer(stated string) (JSONPointer, error) {
	var tokens []string
	for _, escaped := range strings.Split(stated, "/")[1:] {
		tokens = append(tokens, unescapedToken(escaped))
	}
	return JSONPointer{tokens: tokens}, nil
}

func (pointer JSONPointer) Tokens() []string { return pointer.tokens }

// RFC 6901 §4: "~1" is decoded before "~0", so that "~01" yields "~1" and not "/".
func unescapedToken(escaped string) string {
	return strings.ReplaceAll(strings.ReplaceAll(escaped, "~1", "/"), "~0", "~")
}
