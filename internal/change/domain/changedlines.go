// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	hunkHeaderPrefix     = "@@"
	hunkBaseRangeOpen    = hunkHeaderPrefix + " -"
	hunkHeadRangeOpen    = " +"
	hunkHeaderClose      = " " + hunkHeaderPrefix
	hunkSectionOpen      = " "
	hunkRangeJoin        = ","
	hunkNumberBase       = 10
	hunkNumberBits       = 31
	patchLineBreak       = "\n"
	removedLineMark      = '-'
	addedLineMark        = '+'
	firstPatchLineNumber = 1
	unstatedHunkCount    = 1
)

var ErrMalformedHunk = errors.New("the patch holds a malformed hunk")

type ChangedLines struct {
	base []int
	head []int
}

func ParseChangedLines(patch string) (ChangedLines, error) {
	var walk patchWalk
	for index, line := range strings.Split(patch, patchLineBreak) {
		if err := walk.step(line); err != nil {
			return ChangedLines{}, fmt.Errorf("%w: patch line %d", err, index+firstPatchLineNumber)
		}
	}
	return walk.changed, nil
}

type patchWalk struct {
	baseLine int
	headLine int
	isInHunk bool
	changed  ChangedLines
}

func (walk *patchWalk) step(line string) error {
	if strings.HasPrefix(line, hunkHeaderPrefix) {
		return walk.enter(line)
	}
	if !walk.isInHunk || line == "" {
		return ErrMalformedHunk
	}
	switch line[0] {
	case removedLineMark:
		walk.changed.base = append(walk.changed.base, walk.baseLine)
		walk.baseLine++
	case addedLineMark:
		walk.changed.head = append(walk.changed.head, walk.headLine)
		walk.headLine++
	case ' ':
		walk.baseLine++
		walk.headLine++
	default:
		return ErrMalformedHunk
	}
	return nil
}

func (walk *patchWalk) enter(header string) error {
	hunk, err := parseHunkHeader(header)
	if err != nil {
		return err
	}
	walk.baseLine, walk.headLine = hunk.base.start, hunk.head.start
	walk.isInHunk = true
	return nil
}

type hunkHeader struct {
	base hunkRange
	head hunkRange
}

type hunkRange struct {
	start int
	count int
}

func parseHunkHeader(header string) (hunkHeader, error) {
	afterOpen, isOpened := strings.CutPrefix(header, hunkBaseRangeOpen)
	baseRange, afterBase, isBaseClosed := strings.Cut(afterOpen, hunkHeadRangeOpen)
	headRange, section, isHeaderClosed := strings.Cut(afterBase, hunkHeaderClose)
	if !isOpened || !isBaseClosed || !isHeaderClosed || !isHunkSection(section) {
		return hunkHeader{}, ErrMalformedHunk
	}
	base, err := parseHunkRange(baseRange)
	if err != nil {
		return hunkHeader{}, err
	}
	head, err := parseHunkRange(headRange)
	if err != nil {
		return hunkHeader{}, err
	}
	return hunkHeader{base: base, head: head}, nil
}

func isHunkSection(section string) bool {
	return section == "" || strings.HasPrefix(section, hunkSectionOpen)
}

func parseHunkRange(stated string) (hunkRange, error) {
	start, count, hasCount := strings.Cut(stated, hunkRangeJoin)
	parsedStart, err := parseHunkNumber(start)
	if err != nil {
		return hunkRange{}, err
	}
	if !hasCount {
		return hunkRange{start: parsedStart, count: unstatedHunkCount}, nil
	}
	parsedCount, err := parseHunkNumber(count)
	if err != nil {
		return hunkRange{}, err
	}
	return hunkRange{start: parsedStart, count: parsedCount}, nil
}

func parseHunkNumber(stated string) (int, error) {
	number, err := strconv.ParseUint(stated, hunkNumberBase, hunkNumberBits)
	if err != nil {
		return 0, ErrMalformedHunk
	}
	return int(number), nil
}
