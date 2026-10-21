// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "strings"

type JSONPointer struct {
	tokens []string
}

func ParseJSONPointer(stated string) (JSONPointer, error) {
	var tokens []string
	for _, raw := range strings.Split(stated, "/")[1:] {
		tokens = append(tokens, strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~"))
	}
	return JSONPointer{tokens: tokens}, nil
}

func (pointer JSONPointer) Tokens() []string { return pointer.tokens }
