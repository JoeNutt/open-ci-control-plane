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
