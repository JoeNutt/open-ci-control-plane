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

func changeSourceOver(locator domain.PullRequestLocator, files ...domain.File) memory.ChangeSource {
	return memory.ChangeSource{
		Changes: map[domain.PullRequestLocator]memory.Change{
			locator: {
				PullRequest: domain.PullRequest{
					Number:     locator.Number(),
					Repository: locator.Repository(),
					HeadSHA:    recordedHeadSHA,
				},
				Files: files,
			},
		},
	}
}

func TestBuildDocumentCarriesWhatTheForgeReports(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	source := changeSourceOver(locator,
		domain.File{Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
			Additions: 12, Deletions: 3, PatchBytes: 1840},
		domain.File{Path: "infra/iam.tf", Action: domain.FileActionAdded,
			Additions: 35, PatchBytes: 7280},
	)

	useCase := application.BuildDocument{
		Source:    source,
		Contents:  recordedAtHead(t, locator, "internal/auth/middleware.go", sourceImportingContext),
		Extractor: golang.Extractor{},
		Limits:    domain.DeclaredLimits(),
	}

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

func TestBuildDocumentPublishesTheImportsAChangedGoFileDeclaresAtHead(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	file := domain.File{Path: "internal/auth/middleware.go", Action: domain.FileActionModified}
	useCase := application.BuildDocument{
		Source: changeSourceOver(locator, file),
		Contents: recordedAtHead(t, locator, file.Path,
			"package auth\n\nimport (\n\t\"context\"\n\n\t\"github.com/acme/payments/internal/session\"\n)\n"),
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

func TestBuildDocumentReadsNoFileTheExtractorDoesNotClaim(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	useCase := application.BuildDocument{
		Source: changeSourceOver(locator,
			domain.File{Path: "infra/iam.tf", Action: domain.FileActionModified},
			domain.File{Path: "README.md", Action: domain.FileActionAdded},
		),
		Contents:  memory.FileContents{},
		Extractor: golang.Extractor{},
		Limits:    domain.DeclaredLimits(),
	}

	document, err := useCase.Execute(context.Background(), locator)

	if err != nil {
		t.Fatalf("build document: %v", err)
	}
	if len(document.Diff.Imports) != 0 {
		t.Fatalf("files other than Go published imports: %+v", document.Diff.Imports)
	}
}

func TestBuildDocumentReadsNothingAtHeadForARemovedGoFile(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	file := domain.File{Path: "internal/auth/legacy.go", Action: domain.FileActionRemoved}
	useCase := application.BuildDocument{
		Source:    changeSourceOver(locator, file),
		Contents:  memory.FileContents{},
		Extractor: golang.Extractor{},
		Limits:    domain.DeclaredLimits(),
	}

	document, err := useCase.Execute(context.Background(), locator)

	if err != nil {
		t.Fatalf("build document: %v", err)
	}
	if len(document.Diff.Imports) != 0 {
		t.Fatalf("a removed file published head imports: %+v", document.Diff.Imports)
	}
}
