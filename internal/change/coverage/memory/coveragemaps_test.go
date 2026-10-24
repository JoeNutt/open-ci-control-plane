// SPDX-License-Identifier: GPL-3.0-or-later

package memory_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/memory"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	governedRepository = "JoeNutt/open-ci-control-plane"
	otherRepository    = "JoeNutt/open-ci-policy"
	domainSuite        = "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	lcovSuite          = "github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/lcov"
	effectFile         = "internal/change/domain/effect.go"
	oneSuite           = 1
)

func TestCoverageMapsServeTheMapASuiteWasReplacedWith(t *testing.T) {
	var maps domain.CoverageMaps = memory.NewCoverageMaps(domain.DeclaredCoverageLimits())
	repository, suite := parsedRepository(t, governedRepository), parsedSuite(t, domainSuite)
	mustReplace(t, maps, repository, suite)

	served, err := maps.SuiteCoverageOf(t.Context(), repository, suite)
	if err != nil {
		t.Fatalf("serve the suite's map: %v", err)
	}
	if !served.Covers(parsedPath(t, effectFile)) {
		t.Fatalf("served a map that does not cover %s", effectFile)
	}
}

func TestCoverageMapsNeverServeOneRepositorysMapForAnother(t *testing.T) {
	maps := memory.NewCoverageMaps(domain.DeclaredCoverageLimits())
	suite := parsedSuite(t, domainSuite)
	mustReplace(t, maps, parsedRepository(t, governedRepository), suite)

	_, err := maps.SuiteCoverageOf(t.Context(), parsedRepository(t, otherRepository), suite)

	if !errors.Is(err, domain.ErrNoCoverageMap) {
		t.Fatalf("want ErrNoCoverageMap for another repository, got %v", err)
	}
}

func TestCoverageMapsStartEmpty(t *testing.T) {
	maps := memory.NewCoverageMaps(domain.DeclaredCoverageLimits())

	_, err := maps.SuiteCoverageOf(t.Context(), parsedRepository(t, governedRepository), parsedSuite(t, domainSuite))

	if !errors.Is(err, domain.ErrNoCoverageMap) {
		t.Fatalf("want ErrNoCoverageMap from a new store, got %v", err)
	}
}

func TestCoverageMapsKeepServingWhatIsHeldWhenAReplaceIsRefused(t *testing.T) {
	maps := memory.NewCoverageMaps(domain.CoverageLimits{MaxSuitesPerRepository: oneSuite})
	repository, held := parsedRepository(t, governedRepository), parsedSuite(t, domainSuite)
	mustReplace(t, maps, repository, held)

	err := maps.ReplaceSuiteCoverage(t.Context(), repository, parsedSuite(t, lcovSuite), coverageOf(t, effectFile))

	if !errors.Is(err, domain.ErrTooManySuites) {
		t.Fatalf("want ErrTooManySuites, got %v", err)
	}
	if _, err := maps.SuiteCoverageOf(t.Context(), repository, held); err != nil {
		t.Fatalf("the held map stopped serving after a refused replace: %v", err)
	}
}

func mustReplace(t *testing.T, maps domain.CoverageMaps, repository domain.GovernedRepository, suite domain.CoverageSuite) {
	t.Helper()

	if err := maps.ReplaceSuiteCoverage(t.Context(), repository, suite, coverageOf(t, effectFile)); err != nil {
		t.Fatalf("replace %s in %s: %v", suite, repository, err)
	}
}

func coverageOf(t *testing.T, exercised string) domain.SuiteCoverage {
	t.Helper()

	path := parsedPath(t, exercised)
	coverage, err := domain.ParseSuiteCoverage(
		[]domain.ReportedTest{domain.NewReportedTest(effectTestName(t), []domain.TestedFile{domain.NewTestedFile(path, []uint32{1})})},
		domain.NewKnownPaths([]domain.RepositoryPath{path}))
	if err != nil {
		t.Fatalf("parse coverage of %s: %v", exercised, err)
	}
	return coverage
}

func effectTestName(t *testing.T) domain.TopLevelTestName {
	t.Helper()

	name, err := domain.ParseTopLevelTestName("TestEffect")
	if err != nil {
		t.Fatalf("parse test name: %v", err)
	}
	return name
}

func parsedPath(t *testing.T, path string) domain.RepositoryPath {
	t.Helper()

	parsed, err := domain.ParseRepositoryPath(path)
	if err != nil {
		t.Fatalf("parse repository path %q: %v", path, err)
	}
	return parsed
}

func parsedRepository(t *testing.T, repository string) domain.GovernedRepository {
	t.Helper()

	parsed, err := domain.ParseGovernedRepository(repository)
	if err != nil {
		t.Fatalf("parse repository %q: %v", repository, err)
	}
	return parsed
}

func parsedSuite(t *testing.T, suite string) domain.CoverageSuite {
	t.Helper()

	parsed, err := domain.ParseCoverageSuite(suite)
	if err != nil {
		t.Fatalf("parse suite %q: %v", suite, err)
	}
	return parsed
}
