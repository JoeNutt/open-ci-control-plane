// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"strings"
	"unicode"
)

// CommonMark 0.31.2 section 2.4: any ASCII punctuation character may be
// backslash-escaped. Section 2.5: a character reference cannot stand in for a
// character that defines structure. https://spec.commonmark.org/0.31.2/
const markdownASCIIPunctuation = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

func markdownTextOf(stated string) string {
	var encoded strings.Builder
	for _, character := range stated {
		switch {
		case strings.ContainsRune(markdownASCIIPunctuation, character):
			encoded.WriteRune('\\')
			encoded.WriteRune(character)
		case unicode.IsControl(character):
			fmt.Fprintf(&encoded, "&#%d;", character)
		default:
			encoded.WriteRune(character)
		}
	}
	return encoded.String()
}
