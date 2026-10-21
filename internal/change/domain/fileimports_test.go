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

func TestFileImportsInReportsAnExtractorFailureOtherThanUnparsableAsAnError(t *testing.T) {
	failure := errors.New("the extractor ran out of memory")

	_, err := domain.FileImportsIn(emptySource(t), failingExtractor{failure}, domain.DeclaredLimits())

	if !errors.Is(err, failure) {
		t.Fatalf("error is %v, want %v", err, failure)
	}
}

func erroredFileImports(t *testing.T) domain.FileImports {
	t.Helper()

	errored, err := domain.FileImportsIn(
		emptySource(t), failingExtractor{domain.ErrUnparsableSource}, domain.DeclaredLimits())
	if err != nil {
		t.Fatalf("read the imports of a source that does not parse: %v", err)
	}
	return errored
}

func emptySource(t *testing.T) domain.BoundedSource {
	t.Helper()

	source, err := domain.ParseBoundedSource(nil, domain.DeclaredLimits())
	if err != nil {
		t.Fatalf("bound an empty source: %v", err)
	}
	return source
}

type failingExtractor struct{ failure error }

func (failingExtractor) Claims(string) bool { return true }

func (extractor failingExtractor) ImportsIn(domain.BoundedSource) ([]string, error) {
	return nil, extractor.failure
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
