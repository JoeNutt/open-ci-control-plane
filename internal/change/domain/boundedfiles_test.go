// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func withinBounds(t *testing.T, files ...domain.File) domain.BoundedFiles {
	t.Helper()

	limits := domain.Limits{MaxChangedFiles: len(files)}
	for _, file := range files {
		limits.MaxPatchBytesPerFile = max(limits.MaxPatchBytesPerFile, file.PatchBytes)
	}

	bounded, err := domain.ParseBoundedFiles(files, limits)
	if err != nil {
		t.Fatalf("parse bounded files: %v", err)
	}
	return bounded
}

func TestParseBoundedFilesRefusesMoreChangedFilesThanTheLimitAllows(t *testing.T) {
	limits := domain.Limits{MaxChangedFiles: 2}
	files := []domain.File{
		{Path: "internal/auth/middleware.go", Action: domain.FileActionModified},
		{Path: "internal/http/routes.go", Action: domain.FileActionModified},
		{Path: "infra/iam.tf", Action: domain.FileActionAdded},
	}

	_, err := domain.ParseBoundedFiles(files, limits)

	if !errors.Is(err, domain.ErrTooManyChangedFiles) {
		t.Fatalf("error is %v, want %v", err, domain.ErrTooManyChangedFiles)
	}
}

func TestParseBoundedFilesRefusesAFilePatchLargerThanTheLimitAllows(t *testing.T) {
	limits := domain.Limits{MaxChangedFiles: 1, MaxPatchBytesPerFile: 1840}
	files := []domain.File{
		{Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
			PatchBytes: 1841},
	}

	_, err := domain.ParseBoundedFiles(files, limits)

	if !errors.Is(err, domain.ErrFilePatchTooLarge) {
		t.Fatalf("error is %v, want %v", err, domain.ErrFilePatchTooLarge)
	}
}

func TestParseBoundedFilesDoesNotAliasTheSuppliedFileRecords(t *testing.T) {
	files := []domain.File{
		{Path: "internal/auth/middleware.go", Action: domain.FileActionModified},
	}

	bounded := withinBounds(t, files...)
	files[0] = domain.File{Path: "infra/iam.tf", Action: domain.FileActionAdded}

	document := domain.ParseDocument(domain.PullRequest{}, bounded)
	if document.Diff.Files[0].Path != "internal/auth/middleware.go" {
		t.Fatalf("the bounded files followed a later write to their input: %+v",
			document.Diff.Files[0])
	}
}
