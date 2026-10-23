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
	reported   map[domain.CommitSHA]bool
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

func (ledger *plannedHeads) RecordDispatch(_ context.Context, dispatched domain.DispatchedPlan) error {
	if ledger.dispatched == nil {
		ledger.dispatched = map[domain.CommitSHA]domain.DispatchedPlan{}
	}
	ledger.dispatched[dispatched.Head] = dispatched
	return nil
}

func ledgerHolding(t *testing.T, plan domain.TestPlan) *plannedHeads {
	t.Helper()

	return &plannedHeads{dispatched: map[domain.CommitSHA]domain.DispatchedPlan{
		plan.Head: domain.DispatchedPlanOf(gatedLocator(t), plan),
	}}
}

func (ledger *plannedHeads) RecordReport(_ context.Context, _ domain.GovernedRepository, head domain.CommitSHA) error {
	if ledger.reported[head] {
		return domain.ErrRunAlreadyReported
	}
	if ledger.reported == nil {
		ledger.reported = map[domain.CommitSHA]bool{}
	}
	ledger.reported[head] = true
	return nil
}

func TestAReportForAHeadWithNoDispatchedPlanIsRefusedAndPostsNothing(t *testing.T) {
	published := &publishedCheckRuns{}
	useCase := application.RecordRun{Ledger: &plannedHeads{}, Publisher: published}
	head := commitSHAOf(t, recordedHeadSHA)

	err := useCase.Execute(context.Background(), runReportOf(t, head, nil))

	if !errors.Is(err, domain.ErrNoPlanForHead) {
		t.Errorf("a report for an unplanned head gave %v, want %v", err, domain.ErrNoPlanForHead)
	}
	if runs := published.PublishedOn(gatedLocator(t), head); len(runs) != 0 {
		t.Errorf("posted %d check runs for an unplanned head", len(runs))
	}
}

func runReportOf(t *testing.T, head domain.CommitSHA, results []domain.TestResult) domain.RunReport {
	t.Helper()

	report, err := domain.ParseRunReport(gatedLocator(t).GovernedRepository(), head, results)
	if err != nil {
		t.Fatalf("parse run report: %v", err)
	}
	return report
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
		published := &publishedCheckRuns{}

		err := application.RecordRun{Ledger: ledgerHolding(t, report.plan), Publisher: published}.Execute(
			context.Background(), runReportOf(t, head, report.results))

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

func TestASecondReportForOneHeadIsRefusedAndPostsNothingMore(t *testing.T) {
	head := commitSHAOf(t, recordedHeadSHA)
	suite := plannedSuiteOf(t)
	published := &publishedCheckRuns{}
	useCase := application.RecordRun{
		Ledger:    ledgerHolding(t, domain.TestPlan{Head: head, Suites: []domain.CoverageSuite{suite}}),
		Publisher: published,
	}
	first := runReportOf(t, head, []domain.TestResult{{Suite: suite, Name: "TestReadFrom", Outcome: domain.TestOutcomePass}})
	if err := useCase.Execute(context.Background(), first); err != nil {
		t.Fatalf("record the first run: %v", err)
	}
	second := runReportOf(t, head, []domain.TestResult{{Suite: suite, Name: "TestReadFrom", Outcome: domain.TestOutcomeFail}})

	err := useCase.Execute(context.Background(), second)

	if !errors.Is(err, domain.ErrRunAlreadyReported) {
		t.Errorf("a second report for one head gave %v, want %v", err, domain.ErrRunAlreadyReported)
	}
	if run := published.PublishedOn(gatedLocator(t), head)[domain.TestsCheckRunName]; run.Conclusion != domain.ConclusionSuccess {
		t.Errorf("a refused second report changed the check run to %q", run.Conclusion)
	}
}
