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

	plan.DecisionID = decisionIDOf(t, gatedDecisionID)
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
	useCase := application.RecordRun{Ledger: &plannedHeads{}, Publisher: published, Callers: tokensNaming{gatedLocator(t).GovernedRepository()}}
	head := commitSHAOf(t, recordedHeadSHA)

	err := useCase.Execute(context.Background(), application.RunUpload{Report: runReportOf(t, head, nil)})

	if !errors.Is(err, domain.ErrNoPlanForHead) {
		t.Errorf("a report for an unplanned head gave %v, want %v", err, domain.ErrNoPlanForHead)
	}
	if runs := published.PublishedOn(gatedLocator(t), head); len(runs) != 0 {
		t.Errorf("posted %d check runs for an unplanned head", len(runs))
	}
}

func runReportOf(t *testing.T, head domain.CommitSHA, results []domain.TestResult) domain.RunReport {
	t.Helper()

	report, err := domain.ParseRunReport(gatedLocator(t).GovernedRepository(), head, decisionIDOf(t, gatedDecisionID), results)
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

func wholePackageOfSuite(t *testing.T, suite domain.CoverageSuite) []domain.PackageTests {
	t.Helper()

	pkg, err := domain.ParseGoImportPath(suite.String())
	if err != nil {
		t.Fatalf("parse import path: %v", err)
	}
	return []domain.PackageTests{domain.NewPackageTests(pkg, nil)}
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
	skipped := domain.TestResult{Suite: suite, Name: "TestReadFromAnArchive", Outcome: domain.TestOutcomeSkip}

	for _, report := range []struct {
		description string
		plan        domain.TestPlan
		results     []domain.TestResult
		conclusion  domain.Conclusion
	}{
		{"every planned test passed", domain.TestPlan{Head: head, Tests: wholePackageOfSuite(t, suite)},
			[]domain.TestResult{passed}, domain.ConclusionSuccess},
		{"a planned test was skipped", domain.TestPlan{Head: head, Tests: wholePackageOfSuite(t, suite)},
			[]domain.TestResult{passed, skipped}, domain.ConclusionSuccess},
		{"a planned test failed", domain.TestPlan{Head: head, Tests: wholePackageOfSuite(t, suite)},
			[]domain.TestResult{passed, failed}, domain.ConclusionFailure},
		{"a planned suite reported nothing", domain.TestPlan{Head: head, Tests: append(wholePackageOfSuite(t, suite), wholePackageOfSuite(t, unreported)...)},
			[]domain.TestResult{passed}, domain.ConclusionFailure},
		{"the full matrix reported nothing", domain.TestPlan{Head: head, FullMatrix: true},
			nil, domain.ConclusionFailure},
	} {
		published := &publishedCheckRuns{}

		err := application.RecordRun{Ledger: ledgerHolding(t, report.plan), Publisher: published, Callers: tokensNaming{gatedLocator(t).GovernedRepository()}}.Execute(
			context.Background(), application.RunUpload{Report: runReportOf(t, head, report.results)})

		if err != nil {
			t.Fatalf("%s: record run: %v", report.description, err)
		}
		run, isPosted := published.PublishedOn(gatedLocator(t), head)[domain.TestsCheckRunName]
		if !isPosted || run.Conclusion != report.conclusion {
			t.Errorf("%s: posted %+v (posted %t), want %s concluding %s",
				report.description, run, isPosted, domain.TestsCheckRunName, report.conclusion)
		}
		if run.Title == "" || run.Headline == "" {
			t.Errorf("%s: posted %+v with no title or headline, which the forge refuses", report.description, run)
		}
	}
}

func TestASecondReportForOneHeadIsRefusedAndPostsNothingMore(t *testing.T) {
	head := commitSHAOf(t, recordedHeadSHA)
	suite := plannedSuiteOf(t)
	published := &publishedCheckRuns{}
	useCase := application.RecordRun{
		Ledger:    ledgerHolding(t, domain.TestPlan{Head: head, Tests: wholePackageOfSuite(t, suite)}),
		Publisher: published,
		Callers:   tokensNaming{gatedLocator(t).GovernedRepository()},
	}
	first := runReportOf(t, head, []domain.TestResult{{Suite: suite, Name: "TestReadFrom", Outcome: domain.TestOutcomePass}})
	if err := useCase.Execute(context.Background(), application.RunUpload{Report: first}); err != nil {
		t.Fatalf("record the first run: %v", err)
	}
	second := runReportOf(t, head, []domain.TestResult{{Suite: suite, Name: "TestReadFrom", Outcome: domain.TestOutcomeFail}})

	err := useCase.Execute(context.Background(), application.RunUpload{Report: second})

	if !errors.Is(err, domain.ErrRunAlreadyReported) {
		t.Errorf("a second report for one head gave %v, want %v", err, domain.ErrRunAlreadyReported)
	}
	if run := published.PublishedOn(gatedLocator(t), head)[domain.TestsCheckRunName]; run.Conclusion != domain.ConclusionSuccess {
		t.Errorf("a refused second report changed the check run to %q", run.Conclusion)
	}
}

func TestAReportCarryingAnotherRepositorysTokenIsRefusedPostsNothingAndUsesNoReport(t *testing.T) {
	head := commitSHAOf(t, recordedHeadSHA)
	ledger := ledgerHolding(t, domain.TestPlan{Head: head, Tests: wholePackageOfSuite(t, plannedSuiteOf(t))})
	published := &publishedCheckRuns{}
	stranger, err := domain.ParseGovernedRepository("JoeNutt/another-repository")
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}
	useCase := application.RecordRun{Ledger: ledger, Publisher: published, Callers: tokensNaming{stranger}}

	err = useCase.Execute(context.Background(), application.RunUpload{
		Token: presentedToken(t), Report: runReportOf(t, head, nil),
	})

	if !errors.Is(err, application.ErrForeignRepository) {
		t.Errorf("a report with another repository's token gave %v, want %v", err, application.ErrForeignRepository)
	}
	if runs := published.PublishedOn(gatedLocator(t), head); len(runs) != 0 {
		t.Errorf("posted %d check runs for a refused caller", len(runs))
	}
	if ledger.reported[head] {
		t.Error("a refused caller's report used up the head's one report")
	}
}

func TestAReportUnderAnotherDecisionIsRefusedAsUnplannedAndUsesNoReport(t *testing.T) {
	head := commitSHAOf(t, recordedHeadSHA)
	ledger := ledgerHolding(t, domain.TestPlan{Head: head, DecisionID: decisionIDOf(t, gatedDecisionID)})
	published := &publishedCheckRuns{}
	useCase := application.RecordRun{Ledger: ledger, Publisher: published, Callers: tokensNaming{gatedLocator(t).GovernedRepository()}}
	report, err := domain.ParseRunReport(gatedLocator(t).GovernedRepository(), head,
		decisionIDOf(t, "dec_ffffffffffffffffffffffffffffffff"), nil)
	if err != nil {
		t.Fatalf("parse run report: %v", err)
	}

	err = useCase.Execute(context.Background(), application.RunUpload{Token: presentedToken(t), Report: report})

	if !errors.Is(err, domain.ErrNoPlanForHead) {
		t.Errorf("a report under another decision gave %v, want %v", err, domain.ErrNoPlanForHead)
	}
	if runs := published.PublishedOn(gatedLocator(t), head); len(runs) != 0 || ledger.reported[head] {
		t.Errorf("a report under another decision posted %d check runs (report used: %t)", len(runs), ledger.reported[head])
	}
}

type loggedEscapes struct{ escapes []domain.Escape }

func (logged *loggedEscapes) RecordEscape(_ context.Context, escape domain.Escape) error {
	logged.escapes = append(logged.escapes, escape)
	return nil
}

func TestEachEscapeInAShadowRunIsRecordedAgainstItsMode(t *testing.T) {
	head := commitSHAOf(t, recordedHeadSHA)
	outside, err := domain.ParseCoverageSuite("github.com/JoeNutt/open-ci-control-plane/internal/change/domain")
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	ledger := ledgerHolding(t, domain.TestPlan{
		Head: head, FullMatrix: true,
		Recorded: domain.RecordedSets{IsRecorded: true, Selected: wholePackageOfSuite(t, plannedSuiteOf(t))},
	})
	logged := &loggedEscapes{}
	useCase := application.RecordRun{
		Ledger: ledger, Publisher: &publishedCheckRuns{}, Callers: tokensNaming{gatedLocator(t).GovernedRepository()}, Escapes: logged,
	}
	failed := domain.TestResult{Suite: outside, Name: "TestEffect", Outcome: domain.TestOutcomeFail, Rerun: domain.TestOutcomeFail}

	err = useCase.Execute(context.Background(), application.RunUpload{Token: presentedToken(t), Report: runReportOf(t, head, []domain.TestResult{failed})})

	if err != nil {
		t.Fatalf("record run: %v", err)
	}
	if len(logged.escapes) != 1 || logged.escapes[0].Mode != domain.ShadowedSelection || logged.escapes[0].Test != failed {
		t.Errorf("recorded escapes %+v, want one against selection for %s", logged.escapes, failed.Name)
	}
}

type refusingEscapes struct{}

func (refusingEscapes) RecordEscape(context.Context, domain.Escape) error {
	return errors.New("the shadow history is full")
}

func TestAnEscapeThatCannotBeRecordedLeavesTheReportUnspentAndPostsNothing(t *testing.T) {
	head := commitSHAOf(t, recordedHeadSHA)
	outside, err := domain.ParseCoverageSuite("github.com/JoeNutt/open-ci-control-plane/internal/change/domain")
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	ledger := ledgerHolding(t, domain.TestPlan{
		Head: head, FullMatrix: true,
		Recorded: domain.RecordedSets{IsRecorded: true, Selected: wholePackageOfSuite(t, plannedSuiteOf(t))},
	})
	published := &publishedCheckRuns{}
	useCase := application.RecordRun{
		Ledger: ledger, Publisher: published, Callers: tokensNaming{gatedLocator(t).GovernedRepository()}, Escapes: refusingEscapes{},
	}
	failed := domain.TestResult{Suite: outside, Name: "TestEffect", Outcome: domain.TestOutcomeFail, Rerun: domain.TestOutcomeFail}

	err = useCase.Execute(context.Background(), application.RunUpload{Token: presentedToken(t), Report: runReportOf(t, head, []domain.TestResult{failed})})

	if err == nil || ledger.reported[head] || len(published.PublishedOn(gatedLocator(t), head)) != 0 {
		t.Errorf("an unrecordable escape gave %v, report spent %t, posted %d: want an error, unspent, nothing posted",
			err, ledger.reported[head], len(published.PublishedOn(gatedLocator(t), head)))
	}
}
