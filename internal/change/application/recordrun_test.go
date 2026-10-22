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

func TestAReportWhoseEveryPlannedTestPassedPostsASuccessfulTestsCheckRun(t *testing.T) {
	head := commitSHAOf(t, recordedHeadSHA)
	suite := plannedSuiteOf(t)
	ledger := &plannedHeads{dispatched: map[domain.CommitSHA]domain.DispatchedPlan{
		head: {Locator: gatedLocator(t), Plan: domain.TestPlan{Head: head, Suites: []domain.CoverageSuite{suite}}},
	}}
	published := &publishedCheckRuns{}

	err := application.RecordRun{Ledger: ledger, Publisher: published}.Execute(context.Background(), domain.RunReport{
		Repository: gatedLocator(t).GovernedRepository(),
		Head:       head,
		Results:    []domain.TestResult{{Suite: suite, Name: "TestReadFrom", Outcome: domain.TestOutcomePass}},
	})

	if err != nil {
		t.Fatalf("record run: %v", err)
	}
	run, isPosted := published.PublishedOn(gatedLocator(t), head)[domain.TestsCheckRunName]
	if !isPosted || run.Conclusion != domain.ConclusionSuccess {
		t.Errorf("posted %+v (posted %t), want %s concluding success", run, isPosted, domain.TestsCheckRunName)
	}
}

func TestAReportWithAFailedTestPostsAFailedTestsCheckRun(t *testing.T) {
	head := commitSHAOf(t, recordedHeadSHA)
	suite := plannedSuiteOf(t)
	ledger := &plannedHeads{dispatched: map[domain.CommitSHA]domain.DispatchedPlan{
		head: {Locator: gatedLocator(t), Plan: domain.TestPlan{Head: head, Suites: []domain.CoverageSuite{suite}}},
	}}
	published := &publishedCheckRuns{}

	err := application.RecordRun{Ledger: ledger, Publisher: published}.Execute(context.Background(), domain.RunReport{
		Repository: gatedLocator(t).GovernedRepository(),
		Head:       head,
		Results: []domain.TestResult{
			{Suite: suite, Name: "TestReadFrom", Outcome: domain.TestOutcomePass},
			{Suite: suite, Name: "TestReadFromRefusesAliases", Outcome: domain.TestOutcomeFail},
		},
	})

	if err != nil {
		t.Fatalf("record run: %v", err)
	}
	if run := published.PublishedOn(gatedLocator(t), head)[domain.TestsCheckRunName]; run.Conclusion != domain.ConclusionFailure {
		t.Errorf("a report with a failed test concluded %q, want failure", run.Conclusion)
	}
}

func TestAPlannedSuiteThatReportedNothingFailsTheTestsCheckRun(t *testing.T) {
	head := commitSHAOf(t, recordedHeadSHA)
	suite := plannedSuiteOf(t)
	unreported, err := domain.ParseCoverageSuite("github.com/JoeNutt/open-ci-control-plane/internal/change/domain")
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	ledger := &plannedHeads{dispatched: map[domain.CommitSHA]domain.DispatchedPlan{
		head: {Locator: gatedLocator(t), Plan: domain.TestPlan{Head: head, Suites: []domain.CoverageSuite{suite, unreported}}},
	}}
	published := &publishedCheckRuns{}

	err = application.RecordRun{Ledger: ledger, Publisher: published}.Execute(context.Background(), domain.RunReport{
		Repository: gatedLocator(t).GovernedRepository(),
		Head:       head,
		Results:    []domain.TestResult{{Suite: suite, Name: "TestReadFrom", Outcome: domain.TestOutcomePass}},
	})

	if err != nil {
		t.Fatalf("record run: %v", err)
	}
	if run := published.PublishedOn(gatedLocator(t), head)[domain.TestsCheckRunName]; run.Conclusion != domain.ConclusionFailure {
		t.Errorf("a planned suite with no results concluded %q, want failure", run.Conclusion)
	}
}

func TestAnEmptyReportForAFullMatrixPlanFailsTheTestsCheckRun(t *testing.T) {
	head := commitSHAOf(t, recordedHeadSHA)
	ledger := &plannedHeads{dispatched: map[domain.CommitSHA]domain.DispatchedPlan{
		head: {Locator: gatedLocator(t), Plan: domain.TestPlan{Head: head, FullMatrix: true}},
	}}
	published := &publishedCheckRuns{}

	err := application.RecordRun{Ledger: ledger, Publisher: published}.Execute(context.Background(), domain.RunReport{
		Repository: gatedLocator(t).GovernedRepository(), Head: head,
	})

	if err != nil {
		t.Fatalf("record run: %v", err)
	}
	if run := published.PublishedOn(gatedLocator(t), head)[domain.TestsCheckRunName]; run.Conclusion != domain.ConclusionFailure {
		t.Errorf("an empty report for the full matrix concluded %q, want failure", run.Conclusion)
	}
}
