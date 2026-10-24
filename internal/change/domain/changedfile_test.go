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
