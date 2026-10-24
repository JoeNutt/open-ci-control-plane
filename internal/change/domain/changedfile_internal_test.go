// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"slices"
	"testing"
)

func TestChangedFileOfARenameCarriesItsPreviousPathAndItsPatchLines(t *testing.T) {
	stated := StatedFile{
		File:         File{Path: "internal/auth/session.go", Action: FileActionRenamed},
		PreviousPath: "internal/auth/token.go", Patch: "@@ -1,1 +1,1 @@\n-old\n+new", IsPatchStated: true,
	}

	changed, err := ParseChangedFile(stated)

	if err != nil {
		t.Fatalf("parse changed file: %v", err)
	}
	if changed.path.String() != stated.File.Path || changed.base.path.String() != stated.PreviousPath || !changed.base.isAtBase {
		t.Fatalf("paths are %s from %+v, want %s from %s",
			changed.path, changed.base, stated.File.Path, stated.PreviousPath)
	}
	if !changed.lines.isKnown || !slices.Equal(changed.lines.base, []int{1}) || !slices.Equal(changed.lines.head, []int{1}) {
		t.Fatalf("lines are %+v, want known base [1] head [1]", changed.lines)
	}
}

func TestChangedFileThatIsNoRenameSitsAtBaseUnderItsOwnPath(t *testing.T) {
	changed, err := ParseChangedFile(StatedFile{File: File{Path: "internal/auth/owner.go", Action: FileActionModified}})

	if err != nil {
		t.Fatalf("parse changed file: %v", err)
	}
	if changed.base != (basePath{path: changed.path, isAtBase: true}) {
		t.Fatalf("base path is %+v, want %s at base", changed.base, changed.path)
	}
}

func TestChangedFileWhosePatchTheForgeOmitsHasUnknownLines(t *testing.T) {
	stated := StatedFile{File: File{Path: "assets/logo.png", Action: FileActionModified}}

	changed, err := ParseChangedFile(stated)

	if err != nil {
		t.Fatalf("parse changed file: %v", err)
	}
	if changed.lines.isKnown {
		t.Fatalf("lines are %+v, want unknown", changed.lines)
	}
}
