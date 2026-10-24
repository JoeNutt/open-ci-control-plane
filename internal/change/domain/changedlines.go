// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
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

var ErrMalformedHunk = errors.New("the patch holds a malformed hunk")

type ChangedLines struct {
	base []int
	head []int
}

func ParseChangedLines(patch string) (ChangedLines, error) {
	var walk patchWalk
	for _, line := range strings.Split(patch, patchLineBreak) {
		if err := walk.step(line); err != nil {
			return ChangedLines{}, err
		}
	}
	return walk.changed, nil
}

type patchWalk struct {
	baseLine int
	headLine int
	changed  ChangedLines
}

func (walk *patchWalk) step(line string) error {
	if strings.HasPrefix(line, hunkHeaderPrefix) {
		var err error
		walk.baseLine, walk.headLine, err = hunkStartsOf(line)
		return err
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
	return nil
}

func hunkStartsOf(header string) (base, head int, err error) {
	rest, ok := strings.CutPrefix(header, "@@ -")
	if !ok {
		return 0, 0, ErrMalformedHunk
	}
	baseRange, rest, ok := strings.Cut(rest, " +")
	if !ok {
		return 0, 0, ErrMalformedHunk
	}
	headRange, rest, ok := strings.Cut(rest, " @@")
	if !ok {
		return 0, 0, ErrMalformedHunk
	}
	if rest != "" && !strings.HasPrefix(rest, " ") {
		return 0, 0, ErrMalformedHunk
	}
	base, err = rangeStartOf(baseRange)
	if err != nil {
		return 0, 0, err
	}
	head, err = rangeStartOf(headRange)
	if err != nil {
		return 0, 0, err
	}
	return base, head, nil
}

func rangeStartOf(stated string) (int, error) {
	start, count, hasCount := strings.Cut(stated, hunkRangeJoin)
	line, err := strconv.ParseUint(start, 10, 31)
	if err != nil {
		return 0, ErrMalformedHunk
	}
	if hasCount {
		if _, err := strconv.ParseUint(count, 10, 31); err != nil {
			return 0, ErrMalformedHunk
		}
	}
	return int(line), nil
}
