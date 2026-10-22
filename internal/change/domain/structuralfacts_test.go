// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
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
	padding := limits.MaxStructuralBytes - serialisedSizeOf(t, filesImportingPadded(t, 0))

	_, err := domain.ParseStructuralFacts(filesImportingPadded(t, padding+1), limits)

	if !errors.Is(err, domain.ErrStructuralFactsTooLarge) {
		t.Fatalf("error is %v, want %v", err, domain.ErrStructuralFactsTooLarge)
	}
}

func TestParseStructuralFactsAdmitsFactsAtTheSerialisedLimitWhole(t *testing.T) {
	limits := domain.DeclaredLimits()
	padding := limits.MaxStructuralBytes - serialisedSizeOf(t, filesImportingPadded(t, 0))

	_, err := domain.ParseStructuralFacts(filesImportingPadded(t, padding), limits)

	if err != nil {
		t.Fatalf("parse facts serialising to %d bytes: %v", limits.MaxStructuralBytes, err)
	}
}

const (
	paddedImportCount  = 10000
	paddedImportPrefix = "example.com/"
	paddingByte        = "x"
)

func filesImportingPadded(t *testing.T, padding int) []domain.ImportedFile {
	t.Helper()

	written := make([]string, 0, paddedImportCount)
	for index := range paddedImportCount {
		suffix := fmt.Sprintf("p%d", index)
		room := domain.MaxGoImportPathBytes - len(paddedImportPrefix) - len(suffix)
		taken := min(room, padding)
		padding -= taken
		written = append(written, paddedImportPrefix+strings.Repeat(paddingByte, taken)+suffix)
	}
	if padding > 0 {
		t.Fatalf("%d bytes of padding do not fit in %d imports", padding, paddedImportCount)
	}
	return filesKeepingImports(t, written)
}

func serialisedSizeOf(t *testing.T, imported []domain.ImportedFile) int {
	t.Helper()

	unbounded := domain.DeclaredLimits()
	unbounded.MaxStructuralBytes = math.MaxInt
	facts, err := domain.ParseStructuralFacts(imported, unbounded)
	if err != nil {
		t.Fatalf("parse the structural facts of %d files: %v", len(imported), err)
	}
	change := domain.Change{Files: withinBounds(t, changedFilesOf(imported)...), Structure: facts}
	diff := domain.ParseDocument(domain.PullRequest{}, change, domain.ClassificationRules{}).Diff
	return serialisedLengthOf(t, diff.Imports) + serialisedLengthOf(t, diff.ModuleGraphDelta)
}

func serialisedLengthOf[Published domain.Declared[domain.Import] | domain.ModuleGraphDelta](
	t *testing.T, published Published,
) int {
	t.Helper()

	serialised, err := json.Marshal(published)
	if err != nil {
		t.Fatalf("marshal %T: %v", published, err)
	}
	return len(serialised)
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

	return filesKeepingImports(t, numberedImports(total))
}

func filesKeepingImports(t *testing.T, written []string) []domain.ImportedFile {
	t.Helper()

	var imported []domain.ImportedFile
	for declared := range slices.Chunk(written, domain.DeclaredLimits().MaxImportsPerFile) {
		file := domain.File{Path: generatedPath(len(imported)), Action: domain.FileActionModified}
		kept := fileImportsOf(t, declared...)
		imported = append(imported, domain.ImportedFile{File: file, Base: kept, Head: kept})
	}
	return imported
}

func filesAdding(t *testing.T, total int) []domain.ImportedFile {
	t.Helper()

	var imported []domain.ImportedFile
	for declared := range slices.Chunk(numberedImports(total), domain.DeclaredLimits().MaxImportsPerFile) {
		file := domain.File{Path: generatedPath(len(imported)), Action: domain.FileActionAdded}
		imported = append(imported, domain.ImportedFile{File: file, Head: fileImportsOf(t, declared...)})
	}
	return imported
}

func numberedImports(total int) []string {
	written := make([]string, 0, total)
	for index := range total {
		written = append(written, fmt.Sprintf("example.com/module/package%d", index))
	}
	return written
}

func generatedPath(index int) string {
	return fmt.Sprintf("internal/generated/file%d.go", index)
}
