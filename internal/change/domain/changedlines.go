// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"strconv"
	"strings"
)

const (
	hunkHeaderPrefix = "@@"
	patchLineBreak   = "\n"
	removedLineMark  = '-'
	addedLineMark    = '+'
	hunkRangeJoin    = ","
)

type ChangedLines struct {
	base []int
	head []int
}

func ParseChangedLines(patch string) (ChangedLines, error) {
	var walk patchWalk
	for _, line := range strings.Split(patch, patchLineBreak) {
		walk.step(line)
	}
	return walk.changed, nil
}

type patchWalk struct {
	baseLine int
	headLine int
	changed  ChangedLines
}

func (walk *patchWalk) step(line string) {
	if strings.HasPrefix(line, hunkHeaderPrefix) {
		walk.baseLine, walk.headLine = hunkStartsOf(line)
		return
	}
	switch line[0] {
	case removedLineMark:
		walk.changed.base = append(walk.changed.base, walk.baseLine)
		walk.baseLine++
	case addedLineMark:
		walk.changed.head = append(walk.changed.head, walk.headLine)
		walk.headLine++
	default:
		walk.baseLine++
		walk.headLine++
	}
}

func hunkStartsOf(header string) (base, head int) {
	fields := strings.Fields(header)
	return rangeStartOf(fields[1]), rangeStartOf(fields[2])
}

func rangeStartOf(stated string) int {
	start, _, _ := strings.Cut(stated[1:], hunkRangeJoin)
	line, _ := strconv.Atoi(start)
	return line
}
