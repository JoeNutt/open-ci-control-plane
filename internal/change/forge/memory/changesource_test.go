// SPDX-License-Identifier: GPL-3.0-or-later

package memory_test

import (
	"context"
	"errors"
	"reflect"
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
		Changes: map[domain.PullRequestLocator]memory.Change{locator: {Files: files}},
	}

	changed, err := source.FilesChanged(context.Background(), locator)

	if err != nil {
		t.Fatalf("files changed: %v", err)
	}
	if !slices.Equal(changed, files) {
		t.Fatalf("files are %+v, want %+v", changed, files)
	}
}

func TestChangeSourceReportsThePullRequestRecordedForALocator(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	pullRequest := domain.PullRequest{
		Number:     412,
		Repository: "acme/payments",
		BaseRef:    "main",
		HeadSHA:    "9f1c8d2e",
		Author:     domain.Author{Login: "jo", Association: "MEMBER"},
		Labels:     domain.Declared[string]{"backend"},
	}
	source := memory.ChangeSource{
		Changes: map[domain.PullRequestLocator]memory.Change{
			locator: {PullRequest: pullRequest},
		},
	}

	reported, err := source.PullRequestAt(context.Background(), locator)

	if err != nil {
		t.Fatalf("pull request at %s: %v", locator, err)
	}
	if !reflect.DeepEqual(reported, pullRequest) {
		t.Fatalf("pull request is %+v, want %+v", reported, pullRequest)
	}
}

func TestChangeSourceRefusesAPullRequestItHoldsNoFilesFor(t *testing.T) {
	source := memory.ChangeSource{}

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, domain.ErrNoSuchPullRequest) {
		t.Fatalf("error is %v, want %v", err, domain.ErrNoSuchPullRequest)
	}
}
