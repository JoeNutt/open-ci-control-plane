// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"math"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

var noBoundBinds = domain.Limits{
	MaxChangedFiles:      math.MaxInt,
	MaxPatchBytesPerFile: math.MaxInt,
	MaxTotalPatchBytes:   math.MaxInt,
	MaxPathBytes:         math.MaxInt,
	MaxPathSegments:      math.MaxInt,
}

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

	useCase := application.BuildDocument{Source: source, Limits: noBoundBinds}

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
