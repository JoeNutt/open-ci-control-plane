// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/extract/golang"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

func locatorOf(t *testing.T, repository string, number int) domain.PullRequestLocator {
	t.Helper()

	locator, err := domain.ParsePullRequestLocator(repository, number)
	if err != nil {
		t.Fatalf("parse pull request locator: %v", err)
	}
	return locator
}

func TestBuildDocumentCarriesWhatTheForgeReports(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	source := memory.ChangeSource{
		Changes: map[domain.PullRequestLocator]memory.Change{
			locator: {
				PullRequest: domain.PullRequest{Number: 412, Repository: "acme/payments"},
				Files: []domain.File{
					{Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
						Additions: 12, Deletions: 3, PatchBytes: 1840},
					{Path: "infra/iam.tf", Action: domain.FileActionAdded,
						Additions: 35, PatchBytes: 7280},
				},
			},
		},
	}

	useCase := application.BuildDocument{Source: source, Limits: domain.DeclaredLimits()}

	document, err := useCase.Execute(context.Background(), locator)

	if err != nil {
		t.Fatalf("build document: %v", err)
	}
	if document.PR.Number != 412 {
		t.Errorf("pull request number is %d, want 412", document.PR.Number)
	}
	want := domain.Totals{Files: 2, Additions: 47, Deletions: 3, PatchBytes: 9120}
	if document.Diff.Totals != want {
		t.Errorf("totals are %+v, want %+v", document.Diff.Totals, want)
	}
}

const headOfTheChange = "9f1cabc0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6"

func TestBuildDocumentPublishesTheImportsAChangedGoFileDeclaresAtHead(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	file := domain.File{Path: "internal/auth/middleware.go", Action: domain.FileActionModified}
	useCase := application.BuildDocument{
		Source: memory.ChangeSource{
			Changes: map[domain.PullRequestLocator]memory.Change{
				locator: {
					PullRequest: domain.PullRequest{
						Number: 412, Repository: "acme/payments", HeadSHA: headOfTheChange},
					Files: []domain.File{file},
				},
			},
		},
		Contents: memory.FileContents{Files: map[domain.FileAtCommit][]byte{
			{Repository: "acme/payments", Commit: commitOf(t, headOfTheChange), Path: file.Path}: []byte(
				"package auth\n\nimport (\n\t\"context\"\n\n\t\"github.com/acme/payments/internal/session\"\n)\n"),
		}},
		Extractor: golang.Extractor{},
		Limits:    domain.DeclaredLimits(),
	}

	document, err := useCase.Execute(context.Background(), locator)

	if err != nil {
		t.Fatalf("build document: %v", err)
	}
	want := domain.Declared[domain.Import]{{
		Path:  file.Path,
		State: domain.ImportStateResolved,
		Refs:  domain.Declared[string]{"context", "github.com/acme/payments/internal/session"},
	}}
	if !reflect.DeepEqual(document.Diff.Imports, want) {
		t.Fatalf("imports are %+v, want %+v", document.Diff.Imports, want)
	}
}

func commitOf(t *testing.T, stated string) domain.CommitSHA {
	t.Helper()

	commit, err := domain.ParseCommitSHA(stated)
	if err != nil {
		t.Fatalf("parse commit %q: %v", stated, err)
	}
	return commit
}
