// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"slices"
	"testing"
)

func TestChangedLinesOfAModifiedLineNameItOnBothSides(t *testing.T) {
	patch := "@@ -1,3 +1,3 @@\n package auth\n-var limit = 1\n+var limit = 2\n func f() {}"

	lines, err := ParseChangedLines(patch)

	if err != nil {
		t.Fatalf("parse changed lines: %v", err)
	}
	if !slices.Equal(lines.base, []int{2}) || !slices.Equal(lines.head, []int{2}) {
		t.Fatalf("lines are base %v head %v, want base [2] head [2]", lines.base, lines.head)
	}
}

func TestChangedLinesOfAPureInsertionNameTheBaseLinesEitherSideOfIt(t *testing.T) {
	patch := "@@ -1,2 +1,3 @@\n before\n+inserted\n after"

	lines, err := ParseChangedLines(patch)

	if err != nil {
		t.Fatalf("parse changed lines: %v", err)
	}
	if !slices.Equal(lines.base, []int{1, 2}) || !slices.Equal(lines.head, []int{2}) {
		t.Fatalf("lines are base %v head %v, want base [1 2] head [2]", lines.base, lines.head)
	}
}

func TestChangedLinesOfAPureDeletionNameTheHeadLinesEitherSideOfIt(t *testing.T) {
	patch := "@@ -1,3 +1,2 @@\n before\n-deleted\n after"

	lines, err := ParseChangedLines(patch)

	if err != nil {
		t.Fatalf("parse changed lines: %v", err)
	}
	if !slices.Equal(lines.base, []int{2}) || !slices.Equal(lines.head, []int{1, 2}) {
		t.Fatalf("lines are base %v head %v, want base [2] head [1 2]", lines.base, lines.head)
	}
}

func TestChangedLinesOfAnEmptyRangeReadItsStartAsTheLineBeforeTheChange(t *testing.T) {
	patch := "@@ -5,0 +6,1 @@\n+inserted"

	lines, err := ParseChangedLines(patch)

	if err != nil {
		t.Fatalf("parse changed lines: %v", err)
	}
	if !slices.Equal(lines.base, []int{5, 6}) || !slices.Equal(lines.head, []int{6}) {
		t.Fatalf("lines are base %v head %v, want base [5 6] head [6]", lines.base, lines.head)
	}
}

func TestChangedLinesPassOverTheMarkOfAMissingFinalLineBreak(t *testing.T) {
	patch := "@@ -1,1 +1,1 @@\n-old\n\\ No newline at end of file\n+new\n\\ No newline at end of file"

	lines, err := ParseChangedLines(patch)

	if err != nil {
		t.Fatalf("parse changed lines: %v", err)
	}
	if !slices.Equal(lines.base, []int{1}) || !slices.Equal(lines.head, []int{1}) {
		t.Fatalf("lines are base %v head %v, want base [1] head [1]", lines.base, lines.head)
	}
}
