// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseChangedLinesRefusesAMalformedHunkHeader(t *testing.T) {
	for _, header := range []string{
		"@@ -x,1 +1,1 @@",
		"@@ -1,1 @@",
		"@@ -1,1 +1,1",
		"@@ 1,1 +1,1 @@",
		"@@ -1,1 -1,1 @@",
		"@@ -1,y +1,1 @@",
		"@@ -1, +1,1 @@",
		"@@ --1,1 +1,1 @@",
		"@@ -99999999999999999999,1 +1,1 @@",
		"@@",
	} {
		_, err := domain.ParseChangedLines(header + "\n context")

		if !errors.Is(err, domain.ErrMalformedHunk) {
			t.Errorf("%q: error is %v, want %v", header, err, domain.ErrMalformedHunk)
		}
	}
}

func TestParseChangedLinesRefusesAHunkWhoseLinesDisagreeWithItsHeader(t *testing.T) {
	for _, patch := range []string{
		"@@ -1,2 +1,2 @@\n context",
		"@@ -1,1 +1,1 @@\n context\n context",
		"@@ -1,2 +1,1 @@\n context\n+added",
		"@@ -1,2 +1,2 @@\n context\n@@ -9,1 +9,1 @@\n context",
	} {
		_, err := domain.ParseChangedLines(patch)

		if !errors.Is(err, domain.ErrMalformedHunk) {
			t.Errorf("%q: error is %v, want %v", patch, err, domain.ErrMalformedHunk)
		}
	}
}

func TestParseChangedLinesRefusesALineOutsideAnyHunk(t *testing.T) {
	for _, patch := range []string{
		" context before any header\n@@ -1,1 +1,1 @@\n context",
		"@@ -1,2 +1,2 @@\n context\n\n context",
		"@@ -1,2 +1,2 @@\n context\n*starred",
		"",
	} {
		_, err := domain.ParseChangedLines(patch)

		if !errors.Is(err, domain.ErrMalformedHunk) {
			t.Errorf("%q: error is %v, want %v", patch, err, domain.ErrMalformedHunk)
		}
	}
}
