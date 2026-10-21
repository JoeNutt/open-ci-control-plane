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

func TestParseStructuralFactsPublishesEveryImportOfAChangeAtTheLimit(t *testing.T) {
	limits := domain.DeclaredLimits()
	imported := filesDeclaring(t, limits.MaxImportsPerChange)

	facts, err := domain.ParseStructuralFacts(imported, limits)

	if err != nil {
		t.Fatalf("parse %d imports: %v", limits.MaxImportsPerChange, err)
	}
	change := domain.Change{Files: withinBounds(t, changedFilesOf(imported)...), Structure: facts}
	published := 0
	for _, entry := range domain.ParseDocument(domain.PullRequest{}, change, domain.ClassificationRules{}).Diff.Imports {
		published += len(entry.Refs)
	}
	if published != limits.MaxImportsPerChange {
		t.Fatalf("published %d imports, want %d", published, limits.MaxImportsPerChange)
	}
}

func structureOf(t *testing.T, imported ...domain.ImportedFile) domain.StructuralFacts {
	t.Helper()

	facts, err := domain.ParseStructuralFacts(imported, domain.DeclaredLimits())
	if err != nil {
		t.Fatalf("parse the structural facts of %d files: %v", len(imported), err)
	}
	return facts
}

func changedFilesOf(imported []domain.ImportedFile) []domain.File {
	files := make([]domain.File, 0, len(imported))
	for _, file := range imported {
		files = append(files, file.File)
	}
	return files
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
