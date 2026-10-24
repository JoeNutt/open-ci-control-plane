// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"slices"
	"testing"
)

func TestChangedLinesNameWhatAPatchChangesOnEachSide(t *testing.T) {
	for _, stated := range []struct {
		rule  string
		patch string
		base  []int
		head  []int
	}{
		{
			rule:  "a modified line is named on both sides",
			patch: "@@ -1,3 +1,3 @@\n package auth\n-var limit = 1\n+var limit = 2\n func f() {}",
			base:  []int{2}, head: []int{2},
		},
		{
			rule:  "a pure insertion names the base lines either side of it",
			patch: "@@ -1,2 +1,3 @@\n before\n+inserted\n after",
			base:  []int{1, 2}, head: []int{2},
		},
		{
			rule:  "a pure deletion names the head lines either side of it",
			patch: "@@ -1,3 +1,2 @@\n before\n-deleted\n after",
			base:  []int{2}, head: []int{1, 2},
		},
		{
			rule:  "an empty range's start is the line before the change",
			patch: "@@ -5,0 +6,1 @@\n+inserted",
			base:  []int{5, 6}, head: []int{6},
		},
		{
			rule:  "a line named by two blocks is named once",
			patch: "@@ -1,1 +1,3 @@\n+before\n kept\n+after",
			base:  []int{1, 2}, head: []int{1, 3},
		},
		{
			rule:  "a range whose count is omitted spans one line",
			patch: "@@ -3 +3 @@\n-old\n+new",
			base:  []int{3}, head: []int{3},
		},
		{
			rule:  "an empty range at the start of an added file names no base line",
			patch: "@@ -0,0 +1,2 @@\n+one\n+two",
			base:  nil, head: []int{1, 2},
		},
		{
			rule:  "an empty range at the start of a deleted file names no head line",
			patch: "@@ -1,2 +0,0 @@\n-one\n-two",
			base:  []int{1, 2}, head: nil,
		},
		{
			rule:  "the mark of a missing final line break changes neither side",
			patch: "@@ -1,1 +1,1 @@\n-old\n\\ No newline at end of file\n+new\n\\ No newline at end of file",
			base:  []int{1}, head: []int{1},
		},
	} {
		t.Run(stated.rule, func(t *testing.T) {
			lines, err := parseChangedLines(stated.patch)

			if err != nil {
				t.Fatalf("parse changed lines: %v", err)
			}
			if !slices.Equal(lines.base, stated.base) || !slices.Equal(lines.head, stated.head) {
				t.Fatalf("lines are base %v head %v, want base %v head %v",
					lines.base, lines.head, stated.base, stated.head)
			}
		})
	}
}

func TestChangedLinesOfAnEmptyPatchAreKnownAndNone(t *testing.T) {
	lines, err := parseChangedLines("")

	if err != nil || !lines.isKnown || lines.base != nil || lines.head != nil {
		t.Fatalf("lines are %+v with %v, want known and none", lines, err)
	}
}

func TestChangedLinesOfAnUnstatedPatchAreUnknownWhereAParsedPatchIsKnown(t *testing.T) {
	parsed, err := parseChangedLines("@@ -1,1 +1,1 @@\n context")
	if err != nil {
		t.Fatalf("parse changed lines: %v", err)
	}

	if unknownChangedLines().isKnown || !parsed.isKnown {
		t.Fatalf("unknown lines read known %v, parsed lines read known %v; want false and true",
			unknownChangedLines().isKnown, parsed.isKnown)
	}
}
