// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"net/url"
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

const httpsPrefix = "https://"

// CommonMark 0.31.2 section 6.5: an autolink may not contain a space, '<', '>' or
// an ASCII control character. RFC 3986 section 2 lists the characters a URI holds.
const uriCharacters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz" +
	"0123456789-._~:/?#[]@!$&'()*+,;=%"

func markdownHTTPSAutolinkOf(stated string) (string, bool) {
	parsed, err := url.Parse(stated)
	if err != nil || !strings.HasPrefix(stated, httpsPrefix) || parsed.Host == "" {
		return "", false
	}
	if !isURIText(stated) {
		return "", false
	}
	return "<" + stated + ">", true
}

func isURIText(stated string) bool {
	for _, character := range stated {
		if !strings.ContainsRune(uriCharacters, character) {
			return false
		}
	}
	return true
}
