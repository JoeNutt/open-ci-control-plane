// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"strconv"
	"strings"
)

type ChangedLines struct {
	base []int
	head []int
}

func ParseChangedLines(patch string) (ChangedLines, error) {
	var lines ChangedLines
	baseLine, headLine := 0, 0
	for _, line := range strings.Split(patch, "\n") {
		if strings.HasPrefix(line, "@@") {
			fields := strings.Fields(line)
			baseStart, _, _ := strings.Cut(strings.TrimPrefix(fields[1], "-"), ",")
			headStart, _, _ := strings.Cut(strings.TrimPrefix(fields[2], "+"), ",")
			baseLine, _ = strconv.Atoi(baseStart)
			headLine, _ = strconv.Atoi(headStart)
			continue
		}
		switch line[0] {
		case '-':
			lines.base = append(lines.base, baseLine)
			baseLine++
		case '+':
			lines.head = append(lines.head, headLine)
			headLine++
		default:
			baseLine++
			headLine++
		}
	}
	return lines, nil
}
