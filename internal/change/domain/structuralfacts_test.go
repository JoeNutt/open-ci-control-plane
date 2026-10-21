// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseStructuralFactsRefusesOneImportOverTheChangeLimit(t *testing.T) {
	limits := domain.DeclaredLimits()

	_, err := domain.ParseStructuralFacts(filesDeclaring(t, limits.MaxImportsPerChange+1), limits)

	if !errors.Is(err, domain.ErrTooManyImportsInChange) {
		t.Fatalf("error is %v, want %v", err, domain.ErrTooManyImportsInChange)
	}
}

func filesDeclaring(t *testing.T, total int) []domain.ImportedFile {
	t.Helper()

	perFile := domain.DeclaredLimits().MaxImportsPerFile
	var imported []domain.ImportedFile
	for first := 0; first < total; first += perFile {
		file := domain.File{
			Path:   fmt.Sprintf("internal/generated/file%d.go", first/perFile),
			Action: domain.FileActionAdded,
		}
		declared := make([]string, 0, perFile)
		for index := first; index < min(first+perFile, total); index++ {
			declared = append(declared, fmt.Sprintf("example.com/module/package%d", index))
		}
		imported = append(imported, domain.ImportedFile{File: file, Head: fileImportsOf(t, declared...)})
	}
	return imported
}
