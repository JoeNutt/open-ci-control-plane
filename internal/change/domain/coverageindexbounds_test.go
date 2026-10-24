// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestRepositoryCoverageRefusesAnIndexOfMoreTestsThanTheBoundAndKeepsThePrevious(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	full := mustReplace(t, domain.RepositoryCoverage{}, mustParseSuite(t, domainSuite), suiteOfTests(t, limits.MaxIndexedTests))
	oneMore := mustParseSuite(t, lcovSuite)

	_, err := full.Replacing(oneMore, suiteOfTests(t, 1), limits)

	if !errors.Is(err, domain.ErrTooManyIndexedTests) {
		t.Fatalf("want ErrTooManyIndexedTests, got %v", err)
	}
	if _, err := full.SuiteCoverageOf(oneMore); !errors.Is(err, domain.ErrNoCoverageMap) {
		t.Fatalf("the refused suite reached the held coverage: %v", err)
	}
}

func TestRepositoryCoverageRefusesAnIndexOfMoreEdgesThanTheBoundAndKeepsThePrevious(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	held := mustReplace(t, domain.RepositoryCoverage{}, mustParseSuite(t, domainSuite), suiteOfTests(t, 1))
	tooWide := mustParseSuite(t, lcovSuite)

	_, err := held.Replacing(tooWide, suiteOfEdges(t, limits.MaxIndexEdges), limits)

	if !errors.Is(err, domain.ErrTooManyIndexEdges) {
		t.Fatalf("want ErrTooManyIndexEdges, got %v", err)
	}
	if _, err := held.SuiteCoverageOf(tooWide); !errors.Is(err, domain.ErrNoCoverageMap) {
		t.Fatalf("the refused suite reached the held coverage: %v", err)
	}
}

func TestRepositoryCoverageAcceptsAnIndexOfExactlyTheEdgeBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()

	_, err := domain.RepositoryCoverage{}.Replacing(mustParseSuite(t, domainSuite), suiteOfEdges(t, limits.MaxIndexEdges), limits)

	if err != nil {
		t.Fatalf("want an index at the edge bound accepted, got %v", err)
	}
}

func suiteOfEdges(t *testing.T, count int) domain.SuiteCoverage {
	t.Helper()

	held := mustParseRepositoryPath(t, heldFile)
	lines := make([]uint32, 0, count)
	for line := range count {
		lines = append(lines, uint32(line+1))
	}
	tested := domain.NewReportedTest(mustParseTestName(t, effectTest), []domain.TestedFile{domain.NewTestedFile(held, lines)})
	return suiteHoldingOnly(t, held, []domain.ReportedTest{tested})
}

func suiteOfTests(t *testing.T, count int) domain.SuiteCoverage {
	t.Helper()

	held := mustParseRepositoryPath(t, heldFile)
	tests := make([]domain.ReportedTest, 0, count)
	for index := range count {
		name := mustParseTestName(t, fmt.Sprintf("TestGenerated%d", index))
		tests = append(tests, domain.NewReportedTest(name, []domain.TestedFile{domain.NewTestedFile(held, []uint32{1})}))
	}
	return suiteHoldingOnly(t, held, tests)
}

func suiteHoldingOnly(t *testing.T, held domain.RepositoryPath, tests []domain.ReportedTest) domain.SuiteCoverage {
	t.Helper()

	paths := []domain.RepositoryPath{held}
	coverage, err := domain.ParseSuiteCoverage(domain.NewReportedCoverage(paths, tests), domain.NewKnownPaths(paths))
	if err != nil {
		t.Fatalf("parse a suite of %d tests running %s: %v", len(tests), held, err)
	}
	return coverage
}
