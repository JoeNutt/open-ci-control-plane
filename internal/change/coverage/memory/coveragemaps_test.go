// SPDX-License-Identifier: GPL-3.0-or-later

package memory_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/memory"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	governedRepository = "JoeNutt/open-ci-control-plane"
	domainSuite        = "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	effectFile         = "internal/change/domain/effect.go"
)

func TestCoverageMapsServeTheMapASuiteWasReplacedWith(t *testing.T) {
	maps := memory.NewCoverageMaps(domain.DeclaredCoverageLimits())
	repository, suite := parsedRepository(t, governedRepository), parsedSuite(t, domainSuite)

	if err := maps.ReplaceSuiteCoverage(repository, suite, coverageOf(t, effectFile)); err != nil {
		t.Fatalf("replace the suite's map: %v", err)
	}

	served, err := maps.SuiteCoverageOf(repository, suite)
	if err != nil {
		t.Fatalf("serve the suite's map: %v", err)
	}
	if !served.Covers(parsedPath(t, effectFile)) {
		t.Fatalf("served a map that does not cover %s", effectFile)
	}
}

func coverageOf(t *testing.T, exercised string) domain.SuiteCoverage {
	t.Helper()

	path := parsedPath(t, exercised)
	coverage, err := domain.ParseSuiteCoverage(
		[]domain.ReportedFile{domain.NewReportedFile(path, true)},
		domain.NewKnownPaths([]domain.RepositoryPath{path}))
	if err != nil {
		t.Fatalf("parse coverage of %s: %v", exercised, err)
	}
	return coverage
}

func parsedPath(t *testing.T, path string) domain.RepositoryPath {
	t.Helper()

	parsed, err := domain.ParseRepositoryPath(path)
	if err != nil {
		t.Fatalf("parse repository path %q: %v", path, err)
	}
	return parsed
}

func parsedRepository(t *testing.T, repository string) domain.Repository {
	t.Helper()

	parsed, err := domain.ParseRepository(repository)
	if err != nil {
		t.Fatalf("parse repository %q: %v", repository, err)
	}
	return parsed
}

func parsedSuite(t *testing.T, suite string) domain.SuiteName {
	t.Helper()

	parsed, err := domain.ParseSuiteName(suite)
	if err != nil {
		t.Fatalf("parse suite %q: %v", suite, err)
	}
	return parsed
}
