// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type plannedHeads struct {
	dispatched map[domain.CommitSHA]domain.DispatchedPlan
}

func (ledger *plannedHeads) DispatchedPlanFor(
	_ context.Context, _ domain.GovernedRepository, head domain.CommitSHA,
) (domain.DispatchedPlan, error) {
	dispatched, isPlanned := ledger.dispatched[head]
	if !isPlanned {
		return domain.DispatchedPlan{}, domain.ErrNoPlanForHead
	}
	return dispatched, nil
}

func TestAReportForAHeadWithNoDispatchedPlanIsRefusedAndPostsNothing(t *testing.T) {
	published := &publishedCheckRuns{}
	useCase := application.RecordRun{Ledger: &plannedHeads{}, Publisher: published}
	head := commitSHAOf(t, recordedHeadSHA)

	err := useCase.Execute(context.Background(), domain.RunReport{Repository: gatedLocator(t).GovernedRepository(), Head: head})

	if !errors.Is(err, domain.ErrNoPlanForHead) {
		t.Errorf("a report for an unplanned head gave %v, want %v", err, domain.ErrNoPlanForHead)
	}
	if runs := published.PublishedOn(gatedLocator(t), head); len(runs) != 0 {
		t.Errorf("posted %d check runs for an unplanned head", len(runs))
	}
}

const reportedSuite = "github.com/JoeNutt/open-ci-control-plane/internal/change/rules"

func plannedSuiteOf(t *testing.T) domain.CoverageSuite {
	t.Helper()

	suite, err := domain.ParseCoverageSuite(reportedSuite)
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	return suite
}

func TestARecordedRunConcludesTheTestsCheckRunFromItsDispatchedPlan(t *testing.T) {
	head := commitSHAOf(t, recordedHeadSHA)
	suite := plannedSuiteOf(t)
	unreported, err := domain.ParseCoverageSuite("github.com/JoeNutt/open-ci-control-plane/internal/change/domain")
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	passed := domain.TestResult{Suite: suite, Name: "TestReadFrom", Outcome: domain.TestOutcomePass}
	failed := domain.TestResult{Suite: suite, Name: "TestReadFromRefusesAliases", Outcome: domain.TestOutcomeFail}

	for _, report := range []struct {
		description string
		plan        domain.TestPlan
		results     []domain.TestResult
		conclusion  domain.Conclusion
	}{
		{"every planned test passed", domain.TestPlan{Head: head, Suites: []domain.CoverageSuite{suite}},
			[]domain.TestResult{passed}, domain.ConclusionSuccess},
		{"a planned test failed", domain.TestPlan{Head: head, Suites: []domain.CoverageSuite{suite}},
			[]domain.TestResult{passed, failed}, domain.ConclusionFailure},
		{"a planned suite reported nothing", domain.TestPlan{Head: head, Suites: []domain.CoverageSuite{suite, unreported}},
			[]domain.TestResult{passed}, domain.ConclusionFailure},
		{"the full matrix reported nothing", domain.TestPlan{Head: head, FullMatrix: true},
			nil, domain.ConclusionFailure},
	} {
		ledger := &plannedHeads{dispatched: map[domain.CommitSHA]domain.DispatchedPlan{
			head: {Locator: gatedLocator(t), Plan: report.plan},
		}}
		published := &publishedCheckRuns{}

		err := application.RecordRun{Ledger: ledger, Publisher: published}.Execute(context.Background(), domain.RunReport{
			Repository: gatedLocator(t).GovernedRepository(), Head: head, Results: report.results,
		})

		if err != nil {
			t.Fatalf("%s: record run: %v", report.description, err)
		}
		run, isPosted := published.PublishedOn(gatedLocator(t), head)[domain.TestsCheckRunName]
		if !isPosted || run.Conclusion != report.conclusion {
			t.Errorf("%s: posted %+v (posted %t), want %s concluding %s",
				report.description, run, isPosted, domain.TestsCheckRunName, report.conclusion)
		}
	}
}
