// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseStructuralFactsRefusesOneImportOverTheChangeLimit(t *testing.T) {
	limits := domain.DeclaredLimits()

	_, err := domain.ParseStructuralFacts(filesKeeping(t, limits.MaxImportsPerChange+1), limits)

	if !errors.Is(err, domain.ErrTooManyImportsInChange) {
		t.Fatalf("error is %v, want %v", err, domain.ErrTooManyImportsInChange)
	}
}

func TestParseStructuralFactsPublishesEveryImportOfAChangeAtTheLimit(t *testing.T) {
	limits := domain.DeclaredLimits()
	imported := filesKeeping(t, limits.MaxImportsPerChange)

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

func TestParseStructuralFactsRefusesOneEdgeOverTheDeltaLimit(t *testing.T) {
	limits := domain.DeclaredLimits()

	_, err := domain.ParseStructuralFacts(filesAdding(t, limits.MaxGraphEdges+1), limits)

	if !errors.Is(err, domain.ErrTooManyEdges) {
		t.Fatalf("error is %v, want %v", err, domain.ErrTooManyEdges)
	}
}

func TestParseStructuralFactsPublishesEveryEdgeOfAChangeAtTheDeltaLimit(t *testing.T) {
	limits := domain.DeclaredLimits()
	imported := filesAdding(t, limits.MaxGraphEdges)

	facts, err := domain.ParseStructuralFacts(imported, limits)

	if err != nil {
		t.Fatalf("parse %d edges: %v", limits.MaxGraphEdges, err)
	}
	change := domain.Change{Files: withinBounds(t, changedFilesOf(imported)...), Structure: facts}
	delta := domain.ParseDocument(domain.PullRequest{}, change, domain.ClassificationRules{}).Diff.ModuleGraphDelta
	if len(delta.EdgesAdded) != limits.MaxGraphEdges {
		t.Fatalf("published %d edges, want %d", len(delta.EdgesAdded), limits.MaxGraphEdges)
	}
}

func TestParseStructuralFactsRefusesFactsOneByteOverTheSerialisedLimit(t *testing.T) {
	limits := domain.DeclaredLimits()
	padding := limits.MaxStructuralBytes - serialisedSizeOf(t, fileImportingPadded(t, 0))

	_, err := domain.ParseStructuralFacts(
		[]domain.ImportedFile{fileImportingPadded(t, padding+1)}, limits)

	if !errors.Is(err, domain.ErrStructuralFactsTooLarge) {
		t.Fatalf("error is %v, want %v", err, domain.ErrStructuralFactsTooLarge)
	}
}

func fileImportingPadded(t *testing.T, padding int) domain.ImportedFile {
	t.Helper()

	kept := fileImportsOf(t, "example.com/"+strings.Repeat("x", padding))
	file := domain.File{Path: "internal/change/domain/gate.go", Action: domain.FileActionModified}
	return domain.ImportedFile{File: file, Base: kept, Head: kept}
}

func serialisedSizeOf(t *testing.T, imported domain.ImportedFile) int {
	t.Helper()

	unbounded := domain.DeclaredLimits()
	unbounded.MaxStructuralBytes = math.MaxInt
	facts, err := domain.ParseStructuralFacts([]domain.ImportedFile{imported}, unbounded)
	if err != nil {
		t.Fatalf("parse the structural facts of %s: %v", imported.File.Path, err)
	}
	change := domain.Change{Files: withinBounds(t, imported.File), Structure: facts}
	diff := domain.ParseDocument(domain.PullRequest{}, change, domain.ClassificationRules{}).Diff
	published, err := json.Marshal(struct {
		Imports          domain.Declared[domain.Import] `json:"imports"`
		ModuleGraphDelta domain.ModuleGraphDelta        `json:"module_graph_delta"`
	}{diff.Imports, diff.ModuleGraphDelta})
	if err != nil {
		t.Fatalf("marshal the published structural facts: %v", err)
	}
	return len(published)
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

func filesKeeping(t *testing.T, total int) []domain.ImportedFile {
	t.Helper()

	var imported []domain.ImportedFile
	for index, declared := range importPathsInFilesOf(total) {
		file := domain.File{Path: generatedPath(index), Action: domain.FileActionModified}
		kept := fileImportsOf(t, declared...)
		imported = append(imported, domain.ImportedFile{File: file, Base: kept, Head: kept})
	}
	return imported
}

func filesAdding(t *testing.T, total int) []domain.ImportedFile {
	t.Helper()

	var imported []domain.ImportedFile
	for index, declared := range importPathsInFilesOf(total) {
		file := domain.File{Path: generatedPath(index), Action: domain.FileActionAdded}
		imported = append(imported, domain.ImportedFile{File: file, Head: fileImportsOf(t, declared...)})
	}
	return imported
}

func importPathsInFilesOf(total int) [][]string {
	perFile := domain.DeclaredLimits().MaxImportsPerFile
	var files [][]string
	for first := 0; first < total; first += perFile {
		declared := make([]string, 0, perFile)
		for index := first; index < min(first+perFile, total); index++ {
			declared = append(declared, fmt.Sprintf("example.com/module/package%d", index))
		}
		files = append(files, declared)
	}
	return files
}

func generatedPath(index int) string {
	return fmt.Sprintf("internal/generated/file%d.go", index)
}
