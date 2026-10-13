// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"math"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var noBoundBinds = domain.Limits{
	MaxChangedFiles:      math.MaxInt,
	MaxPatchBytesPerFile: math.MaxInt,
	MaxTotalPatchBytes:   math.MaxInt,
	MaxPathBytes:         math.MaxInt,
	MaxPathSegments:      math.MaxInt,
}

func withinBounds(t *testing.T, files ...domain.File) domain.BoundedFiles {
	t.Helper()

	bounded, err := domain.ParseBoundedFiles(files, noBoundBinds)
	if err != nil {
		t.Fatalf("parse bounded files: %v", err)
	}
	return bounded
}

func TestParseBoundedFilesRefusesMoreChangedFilesThanTheLimitAllows(t *testing.T) {
	files := []domain.File{
		{Path: "internal/auth/middleware.go", Action: domain.FileActionModified},
		{Path: "internal/http/routes.go", Action: domain.FileActionModified},
		{Path: "infra/iam.tf", Action: domain.FileActionAdded},
	}
	limits := noBoundBinds
	limits.MaxChangedFiles = len(files) - 1

	_, err := domain.ParseBoundedFiles(files, limits)

	if !errors.Is(err, domain.ErrTooManyChangedFiles) {
		t.Fatalf("error is %v, want %v", err, domain.ErrTooManyChangedFiles)
	}
}

func TestParseBoundedFilesRefusesAFilePatchLargerThanTheLimitAllows(t *testing.T) {
	files := []domain.File{
		{Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
			PatchBytes: 1840},
	}
	limits := noBoundBinds
	limits.MaxPatchBytesPerFile = files[0].PatchBytes - 1

	_, err := domain.ParseBoundedFiles(files, limits)

	if !errors.Is(err, domain.ErrFilePatchTooLarge) {
		t.Fatalf("error is %v, want %v", err, domain.ErrFilePatchTooLarge)
	}
}

func TestParseBoundedFilesRefusesAChangePatchLargerThanTheLimitAllows(t *testing.T) {
	files := []domain.File{
		{Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
			PatchBytes: 1840},
		{Path: "infra/iam.tf", Action: domain.FileActionAdded, PatchBytes: 7280},
	}
	limits := noBoundBinds
	limits.MaxTotalPatchBytes = files[0].PatchBytes + files[1].PatchBytes - 1

	_, err := domain.ParseBoundedFiles(files, limits)

	if !errors.Is(err, domain.ErrChangePatchTooLarge) {
		t.Fatalf("error is %v, want %v", err, domain.ErrChangePatchTooLarge)
	}
}

func TestParseBoundedFilesRefusesAPathLongerThanTheLimitAllows(t *testing.T) {
	path := "internal/auth/middleware.go"
	limits := noBoundBinds
	limits.MaxPathBytes = len(path) - 1
	files := []domain.File{{Path: path, Action: domain.FileActionModified}}

	_, err := domain.ParseBoundedFiles(files, limits)

	if !errors.Is(err, domain.ErrPathTooLong) {
		t.Fatalf("error is %v, want %v", err, domain.ErrPathTooLong)
	}
}

func TestParseBoundedFilesRefusesAPathDeeperThanTheLimitAllows(t *testing.T) {
	limits := noBoundBinds
	limits.MaxPathSegments = 3
	files := []domain.File{
		{Path: "internal/change/domain/boundedfiles.go", Action: domain.FileActionAdded},
	}

	_, err := domain.ParseBoundedFiles(files, limits)

	if !errors.Is(err, domain.ErrPathTooDeep) {
		t.Fatalf("error is %v, want %v", err, domain.ErrPathTooDeep)
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
