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

	document := domain.ParseDocument(domain.PullRequest{}, withinBounds(t, files...))

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

	document := domain.ParseDocument(domain.PullRequest{}, withinBounds(t, files...))

	if !slices.Equal(document.Diff.Files, files) {
		t.Fatalf("files are %+v, want %+v", document.Diff.Files, files)
	}
}

func TestParseDocumentDoesNotAliasTheSuppliedFileRecords(t *testing.T) {
	files := []domain.File{
		{Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
			PatchBytes: 1840},
	}

	document := domain.ParseDocument(domain.PullRequest{}, withinBounds(t, files...))
	files[0] = domain.File{Path: "infra/iam.tf", Action: domain.FileActionAdded}

	if document.Diff.Files[0].Path != "internal/auth/middleware.go" {
		t.Fatalf("the document followed a later write to its input: %+v", document.Diff.Files[0])
	}
}

func TestParseDocumentDoesNotAliasTheSuppliedPullRequest(t *testing.T) {
	pullRequest := domain.PullRequest{
		Number:    412,
		Labels:    domain.Declared[string]{"backend"},
		Approvals: domain.Declared[domain.Approval]{{Login: "sam"}},
	}

	document := domain.ParseDocument(pullRequest, withinBounds(t))

	pullRequest.Labels[0] = "frontend"
	pullRequest.Approvals[0] = domain.Approval{Login: "jo", IsAuthor: true}

	if document.PR.Labels[0] != "backend" {
		t.Errorf("the labels followed a later write to the input: %v", document.PR.Labels)
	}
	if document.PR.Approvals[0] != (domain.Approval{Login: "sam"}) {
		t.Errorf("the approvals followed a later write to the input: %v", document.PR.Approvals)
	}
}

func TestParseDocumentListsEveryChangedPath(t *testing.T) {
	files := []domain.File{
		{Path: "internal/http/routes.go", Action: domain.FileActionModified},
		{Path: "infra/iam.tf", Action: domain.FileActionAdded},
	}

	document := domain.ParseDocument(domain.PullRequest{}, withinBounds(t, files...))

	want := domain.Paths{"infra/iam.tf", "internal/http/routes.go"}
	if !slices.Equal(slices.Sorted(slices.Values(document.Diff.Paths)), want) {
		t.Fatalf("paths are %v, want %v", document.Diff.Paths, want)
	}
}
