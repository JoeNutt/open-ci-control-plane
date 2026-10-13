// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

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
