// SPDX-License-Identifier: GPL-3.0-or-later

package memory_test

import (
	"context"
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
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

func TestChangeSourceReportsTheFilesRecordedForAPullRequest(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	files := []domain.File{
		{Path: "internal/auth/middleware.go", Action: domain.FileActionModified},
	}
	source := memory.ChangeSource{
		Files: map[domain.PullRequestLocator][]domain.File{locator: files},
	}

	changed, err := source.FilesChanged(context.Background(), locator)

	if err != nil {
		t.Fatalf("files changed: %v", err)
	}
	if !slices.Equal(changed, files) {
		t.Fatalf("files are %+v, want %+v", changed, files)
	}
}
