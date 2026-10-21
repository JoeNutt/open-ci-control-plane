// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/memory"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	uploadingRepository = "JoeNutt/open-ci-control-plane"
	uploadedSuite       = "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	coveredFile         = "internal/change/domain/effect.go"
)

func TestReplaceCoverageStoresTheSuitesMap(t *testing.T) {
	maps := memory.NewCoverageMaps(domain.DeclaredCoverageLimits())
	repository, suite, path := parsedRepository(t), parsedSuite(t), parsedPath(t)

	useCase := application.ReplaceCoverage{Maps: maps}
	if err := useCase.Execute(t.Context(), repository, suite, coverageOf(t, path)); err != nil {
		t.Fatalf("replace the suite's coverage: %v", err)
	}

	served, err := maps.SuiteCoverageOf(t.Context(), repository, suite)
	if err != nil {
		t.Fatalf("serve the replaced map: %v", err)
	}
	if !served.Covers(path) {
		t.Fatalf("the served map does not cover %s", path)
	}
}

func parsedRepository(t *testing.T) domain.Repository {
	t.Helper()

	repository, err := domain.ParseRepository(uploadingRepository)
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}
	return repository
}

func parsedSuite(t *testing.T) domain.SuiteName {
	t.Helper()

	suite, err := domain.ParseSuiteName(uploadedSuite)
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	return suite
}

func parsedPath(t *testing.T) domain.RepositoryPath {
	t.Helper()

	path, err := domain.ParseRepositoryPath(coveredFile)
	if err != nil {
		t.Fatalf("parse path: %v", err)
	}
	return path
}

func coverageOf(t *testing.T, path domain.RepositoryPath) domain.SuiteCoverage {
	t.Helper()

	coverage, err := domain.ParseSuiteCoverage(
		[]domain.ReportedFile{domain.NewReportedFile(path, true)},
		domain.NewKnownPaths([]domain.RepositoryPath{path}))
	if err != nil {
		t.Fatalf("parse coverage: %v", err)
	}
	return coverage
}
