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

	parsed, err := domain.ParseFileImports(imports, domain.DeclaredLimits())
	if err != nil {
		t.Fatalf("parse %d imports: %v", len(imports), err)
	}
	return parsed
}

func TestParseFileImportsKeepsEveryImportAtTheLimit(t *testing.T) {
	limits := domain.DeclaredLimits()
	imports := make([]string, limits.MaxImportsPerFile)
	for index := range imports {
		imports[index] = fmt.Sprintf("example.com/module/package%d", index)
	}

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

	_, err := domain.ParseFileImports(make([]string, limits.MaxImportsPerFile+1), limits)

	if !errors.Is(err, domain.ErrTooManyImportsInFile) {
		t.Fatalf("error is %v, want %v", err, domain.ErrTooManyImportsInFile)
	}
}
