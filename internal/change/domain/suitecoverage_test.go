// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseSuiteCoverageCountsEveryReportedFile(t *testing.T) {
	suite := twoFileSuiteOf(t)

	if suite.coverage.FileEntryCount() != suite.reportedCount {
		t.Fatalf("file entry count is %d, want %d", suite.coverage.FileEntryCount(), suite.reportedCount)
	}
}

func TestParseSuiteCoverageRefusesAPathTheRepositoryDoesNotHold(t *testing.T) {
	held := mustParseRepositoryPath(t, heldFile)
	absent := mustParseRepositoryPath(t, absentFile)
	reported := domain.NewReportedCoverage([]domain.RepositoryPath{held, absent}, []domain.ReportedTest{
		domain.NewReportedTest(mustParseTestName(t, effectTest), []domain.TestedFile{domain.NewTestedFile(held, []uint32{1})}),
	})

	_, err := domain.ParseSuiteCoverage(reported, domain.NewKnownPaths([]domain.RepositoryPath{held}))

	if !errors.Is(err, domain.ErrPathOutsideRepository) {
		t.Fatalf("want ErrPathOutsideRepository, got %v", err)
	}
}

type twoFileSuite struct {
	exercised     domain.RepositoryPath
	unexercised   domain.RepositoryPath
	reportedCount int
	coverage      domain.SuiteCoverage
}

func twoFileSuiteOf(t *testing.T) twoFileSuite {
	t.Helper()

	exercised := mustParseRepositoryPath(t, heldFile)
	unexercised := mustParseRepositoryPath(t, otherHeldFile)
	paths := []domain.RepositoryPath{exercised, unexercised}
	reported := domain.NewReportedCoverage(paths, []domain.ReportedTest{
		domain.NewReportedTest(mustParseTestName(t, effectTest), []domain.TestedFile{domain.NewTestedFile(exercised, []uint32{1})}),
	})
	known := domain.NewKnownPaths(paths)

	coverage, err := domain.ParseSuiteCoverage(reported, known)
	if err != nil {
		t.Fatalf("parse coverage of held paths: %v", err)
	}
	return twoFileSuite{
		exercised:     exercised,
		unexercised:   unexercised,
		reportedCount: len(paths),
		coverage:      coverage,
	}
}
