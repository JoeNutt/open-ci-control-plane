// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseChangedFileRefusesARenameThatStatesNoPreviousPath(t *testing.T) {
	stated := domain.StatedFile{File: domain.File{Path: "internal/auth/session.go", Action: domain.FileActionRenamed}}

	_, err := domain.ParseChangedFile(stated)

	if !errors.Is(err, domain.ErrRenameWithoutPreviousPath) {
		t.Fatalf("error is %v, want %v", err, domain.ErrRenameWithoutPreviousPath)
	}
}

func TestParseChangedFileRefusesAPreviousPathOnAFileThatIsNoRename(t *testing.T) {
	stated := domain.StatedFile{
		File:         domain.File{Path: "internal/auth/session.go", Action: domain.FileActionModified},
		PreviousPath: "internal/auth/token.go",
	}

	_, err := domain.ParseChangedFile(stated)

	if !errors.Is(err, domain.ErrPreviousPathOfNoRenameOrCopy) {
		t.Fatalf("error is %v, want %v", err, domain.ErrPreviousPathOfNoRenameOrCopy)
	}
}

func TestParseChangedFileRefusesAPreviousPathOutsideTheRepository(t *testing.T) {
	stated := domain.StatedFile{
		File:         domain.File{Path: "internal/auth/session.go", Action: domain.FileActionRenamed},
		PreviousPath: "../elsewhere/token.go",
	}

	_, err := domain.ParseChangedFile(stated)

	if !errors.Is(err, domain.ErrPathOutsideRepository) {
		t.Fatalf("error is %v, want %v", err, domain.ErrPathOutsideRepository)
	}
}
