// SPDX-License-Identifier: GPL-3.0-or-later

package github

import "strings"

func markdownCodeSpanOf(value string) string {
	longest, run := 0, 0
	for _, character := range value {
		if character == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	fence := strings.Repeat("`", longest+1)
	return fence + " " + value + " " + fence
}
