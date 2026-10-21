// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/memory"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestReplaceCoverageStoresTheSuitesMap(t *testing.T) {
	maps := memory.NewCoverageMaps(domain.DeclaredCoverageLimits())
	repository, err := domain.ParseRepository("JoeNutt/open-ci-control-plane")
	if err != nil {
		t.Fatal(err)
	}
	suite, err := domain.ParseSuiteName("github.com/JoeNutt/open-ci-control-plane/internal/change/domain")
	if err != nil {
		t.Fatal(err)
	}
	path, err := domain.ParseRepositoryPath("internal/change/domain/effect.go")
	if err != nil {
		t.Fatal(err)
	}
	coverage, err := domain.ParseSuiteCoverage(
		[]domain.ReportedFile{domain.NewReportedFile(path, true)},
		domain.NewKnownPaths([]domain.RepositoryPath{path}))
	if err != nil {
		t.Fatal(err)
	}

	useCase := application.ReplaceCoverage{Maps: maps}
	if err := useCase.Execute(t.Context(), repository, suite, coverage); err != nil {
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
