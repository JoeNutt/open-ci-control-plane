// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func fileImportsOf(t *testing.T, imports ...string) domain.FileImports {
	t.Helper()

	parsed, err := domain.ParseFileImports(goImportPathsOf(t, imports...), domain.DeclaredLimits())
	if err != nil {
		t.Fatalf("parse %d imports: %v", len(imports), err)
	}
	return parsed
}

func goImportPathsOf(t *testing.T, written ...string) []domain.GoImportPath {
	t.Helper()

	paths := make([]domain.GoImportPath, 0, len(written))
	for _, stated := range written {
		path, err := domain.ParseGoImportPath(stated)
		if err != nil {
			t.Fatalf("parse import path %q: %v", stated, err)
		}
		paths = append(paths, path)
	}
	return paths
}

func TestParseFileImportsKeepsEveryImportAtTheLimit(t *testing.T) {
	limits := domain.DeclaredLimits()
	written := make([]string, limits.MaxImportsPerFile)
	for index := range written {
		written[index] = fmt.Sprintf("example.com/module/package%d", index)
	}
	imports := goImportPathsOf(t, written...)

	parsed, err := domain.ParseFileImports(imports, limits)

	if err != nil {
		t.Fatalf("parse %d imports: %v", len(imports), err)
	}
	if !slices.Equal(parsed.ImportPaths(), imports) {
		t.Fatalf("kept %d imports, want the %d given", len(parsed.ImportPaths()), len(imports))
	}
}

func TestParseFileImportsRefusesOneImportOverTheLimit(t *testing.T) {
	limits := domain.DeclaredLimits()

	_, err := domain.ParseFileImports(make([]domain.GoImportPath, limits.MaxImportsPerFile+1), limits)

	if !errors.Is(err, domain.ErrTooManyImportsInFile) {
		t.Fatalf("error is %v, want %v", err, domain.ErrTooManyImportsInFile)
	}
}
