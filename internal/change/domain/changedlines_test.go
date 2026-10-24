// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseChangedFileRefusesAPatchWithAMalformedHunkHeader(t *testing.T) {
	refusesEveryPatchAsMalformed(t, []string{
		"@@ -x,1 +1,1 @@\n context",
		"@@ -1,1 @@\n context",
		"@@ -1,1 +1,1\n context",
		"@@ 1,1 +1,1 @@\n context",
		"@@ -1,1 -1,1 @@\n context",
		"@@ -1,y +1,1 @@\n context",
		"@@ -1, +1,1 @@\n context",
		"@@ --1,1 +1,1 @@\n context",
		"@@ -99999999999999999999,1 +1,1 @@\n context",
		"@@\n context",
	})
}

func TestParseChangedFileRefusesAPatchWithAHunkWhoseLinesDisagreeWithItsHeader(t *testing.T) {
	refusesEveryPatchAsMalformed(t, []string{
		"@@ -1,2 +1,2 @@\n context",
		"@@ -1,1 +1,1 @@\n context\n context",
		"@@ -1,2 +1,1 @@\n context\n+added",
		"@@ -1,2 +1,2 @@\n context\n@@ -9,1 +9,1 @@\n context",
	})
}

func TestParseChangedFileRefusesAPatchWithALineOutsideAnyHunk(t *testing.T) {
	refusesEveryPatchAsMalformed(t, []string{
		" context before any header\n@@ -1,1 +1,1 @@\n context",
		"@@ -1,2 +1,2 @@\n context\n\n context",
		"@@ -1,2 +1,2 @@\n context\n*starred",
		"\\ No newline at end of file\n@@ -1,1 +1,1 @@\n context",
		"",
	})
}

func TestParseChangedFileRefusesAPatchWithARangeThatStartsAtZeroYetSpansLines(t *testing.T) {
	refusesEveryPatchAsMalformed(t, []string{
		"@@ -0,1 +1,1 @@\n context",
		"@@ -1,1 +0,1 @@\n context",
		"@@ -0 +1 @@\n context",
	})
}

func TestParseChangedFileRefusesAPatchWithAHunkThatStartsBeforeThePreviousOneEnded(t *testing.T) {
	refusesEveryPatchAsMalformed(t, []string{
		"@@ -5,2 +5,2 @@\n one\n two\n@@ -6,1 +9,1 @@\n three",
		"@@ -5,2 +5,2 @@\n one\n two\n@@ -9,1 +6,1 @@\n three",
	})
}

func refusesEveryPatchAsMalformed(t *testing.T, patches []string) {
	t.Helper()

	for _, patch := range patches {
		_, err := patchedFileParse(patch)

		if !errors.Is(err, domain.ErrMalformedHunk) {
			t.Errorf("%q: error is %v, want %v", patch, err, domain.ErrMalformedHunk)
		}
	}
}

func patchedFileParse(patch string) (domain.ChangedFile, error) {
	return domain.ParseChangedFile(domain.StatedFile{
		File:  domain.File{Path: "internal/auth/owner.go", Action: domain.FileActionModified},
		Patch: patch, IsPatchStated: true,
	})
}
