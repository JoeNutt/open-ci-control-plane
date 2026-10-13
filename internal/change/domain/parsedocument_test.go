// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseDocumentAggregatesTotalsOverChangedFiles(t *testing.T) {
	files := []domain.File{
		{Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
			Additions: 12, Deletions: 3, PatchBytes: 1840},
		{Path: "infra/iam.tf", Action: domain.FileActionAdded,
			Additions: 35, Deletions: 0, PatchBytes: 7280},
	}

	document, err := domain.ParseDocument(domain.PullRequest{}, files)

	if err != nil {
		t.Fatalf("parse document: %v", err)
	}
	want := domain.Totals{Files: 2, Additions: 47, Deletions: 3, PatchBytes: 9120}
	if document.Diff.Totals != want {
		t.Fatalf("totals are %+v, want %+v", document.Diff.Totals, want)
	}
}

func TestParseDocumentCarriesEveryChangedFileRecord(t *testing.T) {
	files := []domain.File{
		{Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
			Additions: 12, Deletions: 3, PatchBytes: 1840},
	}

	document, err := domain.ParseDocument(domain.PullRequest{}, files)

	if err != nil {
		t.Fatalf("parse document: %v", err)
	}
	if !slices.Equal(document.Diff.Files, files) {
		t.Fatalf("files are %+v, want %+v", document.Diff.Files, files)
	}
}

func TestParseDocumentListsEveryChangedPath(t *testing.T) {
	files := []domain.File{
		{Path: "internal/http/routes.go", Action: domain.FileActionModified},
		{Path: "infra/iam.tf", Action: domain.FileActionAdded},
	}

	document, err := domain.ParseDocument(domain.PullRequest{}, files)

	if err != nil {
		t.Fatalf("parse document: %v", err)
	}
	want := domain.Paths{"infra/iam.tf", "internal/http/routes.go"}
	if !slices.Equal(slices.Sorted(slices.Values(document.Diff.Paths)), want) {
		t.Fatalf("paths are %v, want %v", document.Diff.Paths, want)
	}
}
