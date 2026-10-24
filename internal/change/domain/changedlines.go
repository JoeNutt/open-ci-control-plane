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
	contextLineMark      = ' '
	missingLineBreakMark = '\\'
	firstLineNumber      = 1
	unstatedHunkCount    = 1
)

var ErrMalformedHunk = errors.New("the patch holds a malformed hunk")

type ChangedLines struct {
	base []int
	head []int
}

func ParseChangedLines(patch string) (ChangedLines, error) {
	var walk patchWalk
	lines := strings.Split(patch, patchLineBreak)
	for index, line := range lines {
		if err := walk.step(line); err != nil {
			return ChangedLines{}, malformedAt(index, err)
		}
	}
	walk.closeBlock()
	if walk.isHunkOpen() {
		return ChangedLines{}, malformedAt(len(lines)-1, ErrMalformedHunk)
	}
	return walk.changedLines(), nil
}

func malformedAt(index int, err error) error {
	return fmt.Errorf("%w: patch line %d", err, index+firstLineNumber)
}

type patchWalk struct {
	base    patchSide
	head    patchSide
	hasHunk bool
}

type linePresence struct{ onBase, onHead sidePresence }

type sidePresence struct {
	isPresent bool
	isChanged bool
}

var (
	absentFromSide  = sidePresence{}
	keptOnSide      = sidePresence{isPresent: true}
	changedOnSide   = sidePresence{isPresent: true, isChanged: true}
	keptOnBothSides = linePresence{onBase: keptOnSide, onHead: keptOnSide}
	presenceOfLines = map[byte]linePresence{
		removedLineMark:      {onBase: changedOnSide, onHead: absentFromSide},
		addedLineMark:        {onBase: absentFromSide, onHead: changedOnSide},
		contextLineMark:      keptOnBothSides,
		missingLineBreakMark: {onBase: absentFromSide, onHead: absentFromSide},
	}
)

func (walk *patchWalk) step(line string) error {
	if strings.HasPrefix(line, hunkHeaderPrefix) {
		return walk.enter(line)
	}
	if !walk.hasHunk || line == "" {
		return ErrMalformedHunk
	}
	presence, isMarked := presenceOfLines[line[0]]
	if !isMarked {
		return ErrMalformedHunk
	}
	if presence == keptOnBothSides {
		walk.closeBlock()
	}
	if err := walk.base.take(presence.onBase); err != nil {
		return err
	}
	return walk.head.take(presence.onHead)
}

func (walk *patchWalk) enter(header string) error {
	walk.closeBlock()
	if walk.isHunkOpen() {
		return ErrMalformedHunk
	}
	hunk, err := parseHunkHeader(header)
	if err != nil {
		return err
	}
	walk.base.open(hunk.base)
	walk.head.open(hunk.head)
	walk.hasHunk = true
	return nil
}

func (walk *patchWalk) closeBlock() {
	if walk.isInBlock() {
		walk.base.markNeighboursOfUnchangedBlock()
		walk.head.markNeighboursOfUnchangedBlock()
	}
	walk.base.isChangedInBlock = false
	walk.head.isChangedInBlock = false
}

func (walk patchWalk) isInBlock() bool {
	return walk.base.isChangedInBlock || walk.head.isChangedInBlock
}

func (walk patchWalk) isHunkOpen() bool { return walk.base.remaining > 0 || walk.head.remaining > 0 }

func (walk patchWalk) changedLines() ChangedLines {
	return ChangedLines{base: walk.base.changed, head: walk.head.changed}
}

type patchSide struct {
	line             int
	remaining        int
	changed          []int
	isChangedInBlock bool
}

func (side *patchSide) open(stated hunkRange) {
	side.line, side.remaining = stated.firstLine(), stated.count
}

func (side *patchSide) take(presence sidePresence) error {
	if !presence.isPresent {
		return nil
	}
	if side.remaining == 0 {
		return ErrMalformedHunk
	}
	if presence.isChanged {
		side.changed = append(side.changed, side.line)
		side.isChangedInBlock = true
	}
	side.line++
	side.remaining--
	return nil
}

func (side *patchSide) markNeighboursOfUnchangedBlock() {
	if side.isChangedInBlock {
		return
	}
	if side.line > firstLineNumber {
		side.changed = append(side.changed, side.line-1)
	}
	side.changed = append(side.changed, side.line)
}

type hunkHeader struct {
	base hunkRange
	head hunkRange
}

type hunkRange struct {
	start int
	count int
}

func (stated hunkRange) firstLine() int {
	if stated.isEmpty() {
		return stated.start + 1
	}
	return stated.start
}

func (stated hunkRange) isEmpty() bool { return stated.count == 0 }

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
