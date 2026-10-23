// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func reportedRepositoryAndHead(t *testing.T) (domain.GovernedRepository, domain.CommitSHA) {
	t.Helper()

	return governedRepositoryOf(t, "JoeNutt/open-ci-control-plane"), mustParseCommitSHA(t, planHeadSHA)
}

func TestARunReportCarriesItsRepositoryHeadDecisionAndResults(t *testing.T) {
	repository, head := reportedRepositoryAndHead(t)
	id := reportedDecisionOf(t)
	results := []domain.TestResult{{Name: "TestReadFrom", Outcome: domain.TestOutcomePass}}

	report, err := domain.ParseRunReport(repository, head, id, results)

	if err != nil {
		t.Fatalf("parse run report: %v", err)
	}
	if report.Repository() != repository || report.Head() != head || report.DecisionID() != id ||
		!slices.Equal(report.Results(), results) {
		t.Errorf("parsed %+v, want %s at %s under %s with %+v", report, repository, head, id, results)
	}
}

func reportedDecisionOf(t *testing.T) domain.DecisionID {
	t.Helper()

	id, err := domain.ParseDecisionID("dec_0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("parse decision ID: %v", err)
	}
	return id
}

func TestARunReportIsRefusedAboveMaxReportedTests(t *testing.T) {
	repository, head := reportedRepositoryAndHead(t)

	for _, bounded := range []struct {
		count   int
		refusal error
	}{
		{domain.MaxReportedTests, nil},
		{domain.MaxReportedTests + 1, domain.ErrTooManyReportedTests},
	} {
		_, err := domain.ParseRunReport(repository, head, reportedDecisionOf(t), make([]domain.TestResult, bounded.count))

		if !errors.Is(err, bounded.refusal) {
			t.Errorf("a report of %d tests gave %v, want %v", bounded.count, err, bounded.refusal)
		}
	}
}

const reportedTestSuite = "github.com/JoeNutt/open-ci-control-plane/internal/change/rules"

func TestAReportedTestParsesToItsSuiteNameAndOutcome(t *testing.T) {
	parsed, err := domain.ParseTestResult(reportedTestSuite, "TestReadFrom", "pass")

	if err != nil {
		t.Fatalf("parse test result: %v", err)
	}
	if parsed.Suite.String() != reportedTestSuite || parsed.Name != "TestReadFrom" || parsed.Outcome != domain.TestOutcomePass {
		t.Errorf("parsed %+v, want TestReadFrom in %s passing", parsed, reportedTestSuite)
	}
}

func TestAReportedTestIsRefusedUnlessItsOutcomeIsKnown(t *testing.T) {
	for _, outcome := range []string{"", "passed", "PASS", "error"} {
		_, err := domain.ParseTestResult(reportedTestSuite, "TestReadFrom", outcome)

		if !errors.Is(err, domain.ErrMalformedTestResult) {
			t.Errorf("outcome %q gave %v, want %v", outcome, err, domain.ErrMalformedTestResult)
		}
	}
}

func TestASkippedTestParsesAsSkipped(t *testing.T) {
	parsed, err := domain.ParseTestResult(reportedTestSuite, "TestReadFrom", "skip")

	if err != nil || parsed.Outcome != domain.TestOutcomeSkip {
		t.Errorf("a skipped test parsed to %+v (%v), want outcome %q", parsed, err, domain.TestOutcomeSkip)
	}
}

func TestAReportedTestIsRefusedUnlessItsNameIsAGoIdentifier(t *testing.T) {
	for _, name := range []string{"", "Test Read", "TestReadFrom/aliases", "../TestReadFrom", "TestRéad", "1Test", "Test\nRead"} {
		_, err := domain.ParseTestResult(reportedTestSuite, name, "pass")

		if !errors.Is(err, domain.ErrMalformedTestResult) {
			t.Errorf("name %q gave %v, want %v", name, err, domain.ErrMalformedTestResult)
		}
	}
}

func TestAReportedTestIsRefusedUnlessItsPackageIsASuite(t *testing.T) {
	for _, suite := range []string{"", "../rules", "github.com/JoeNutt/open ci"} {
		_, err := domain.ParseTestResult(suite, "TestReadFrom", "pass")

		if !errors.Is(err, domain.ErrMalformedTestResult) {
			t.Errorf("package %q gave %v, want %v", suite, err, domain.ErrMalformedTestResult)
		}
	}
}

func TestADispatchedPlanKeepsWhatItsPlanRecordedAndAlwaysRuns(t *testing.T) {
	suite, err := domain.ParseCoverageSuite(reportedTestSuite)
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	alwaysRun, err := domain.ParseGoImportPath("github.com/JoeNutt/open-ci-control-plane/internal/change/domain")
	if err != nil {
		t.Fatalf("parse import path: %v", err)
	}
	recorded := domain.RecordedSets{IsRecorded: true, Selected: []domain.CoverageSuite{suite}}

	dispatched := domain.DispatchedPlanOf(domain.PullRequestLocator{}, domain.TestPlan{Recorded: recorded, AlwaysRun: []domain.GoImportPath{alwaysRun}})

	if !dispatched.Recorded.IsRecorded || !slices.Equal(dispatched.Recorded.Selected, recorded.Selected) ||
		!slices.Equal(dispatched.AlwaysRun, []domain.GoImportPath{alwaysRun}) {
		t.Errorf("the dispatched plan keeps %+v and always runs %v, want %+v and %v",
			dispatched.Recorded, dispatched.AlwaysRun, recorded, alwaysRun)
	}
}
