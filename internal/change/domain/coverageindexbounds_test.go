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

func suiteOfTests(t *testing.T, count int) domain.SuiteCoverage {
	t.Helper()

	held := mustParseRepositoryPath(t, heldFile)
	tests := make([]domain.ReportedTest, 0, count)
	for index := range count {
		name := mustParseTestName(t, fmt.Sprintf("TestGenerated%d", index))
		tests = append(tests, domain.NewReportedTest(name, []domain.TestedFile{domain.NewTestedFile(held, []uint32{1})}))
	}
	coverage, err := domain.ParseSuiteCoverage(tests, domain.NewKnownPaths([]domain.RepositoryPath{held}))
	if err != nil {
		t.Fatalf("parse a suite of %d tests: %v", count, err)
	}
	return coverage
}
