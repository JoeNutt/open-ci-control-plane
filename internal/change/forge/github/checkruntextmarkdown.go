// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"fmt"
	"strings"
	"unicode"
)

// CommonMark 0.31.2 section 6.1: "A code span begins with a backtick string and
// ends with a backtick string of equal length", "backslash escapes do not work in
// code spans", and if the content "both begins and ends with a space character ...
// a single space character is removed from the front and back". Section 2.5:
// "Entity and character references are not recognized in code blocks and code
// spans." https://spec.commonmark.org/0.31.2/#code-spans
func markdownCodeSpanOf(value string) string {
	visible := markdownVisibleTextOf(markdownSingleLineOf(value))
	fence := strings.Repeat("`", markdownLongestBacktickRunIn(visible)+1)
	return fence + " " + visible + " " + fence
}

var markdownLineTerminators = strings.NewReplacer(
	"\r\n", " ",
	"\n", " ",
	"\r", " ",
	"\v", " ",
	"\f", " ",
	"\u0085", " ",
	"\u2028", " ",
	"\u2029", " ",
)

func markdownSingleLineOf(value string) string {
	return markdownLineTerminators.Replace(value)
}

const markdownHiddenCharacterMarker = "<U+%04X>"

func markdownVisibleTextOf(line string) string {
	var visible strings.Builder
	for _, character := range line {
		if markdownIsHidden(character) {
			fmt.Fprintf(&visible, markdownHiddenCharacterMarker, character)
			continue
		}
		visible.WriteRune(character)
	}
	return visible.String()
}

func markdownIsHidden(character rune) bool {
	return unicode.Is(unicode.Cf, character) || unicode.IsControl(character)
}

func markdownLongestBacktickRunIn(value string) int {
	longest, run := 0, 0
	for _, character := range value {
		if character != '`' {
			run = 0
			continue
		}
		run++
		longest = max(longest, run)
	}
	return longest
}
