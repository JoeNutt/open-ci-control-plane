// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/history/jsonl"
)

const (
	ledgerRepository  = "JoeNutt/open-ci-control-plane"
	ledgerHeadSHA     = "9f1cabc0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6"
	ledgerDecisionID  = "dec_0123456789abcdef0123456789abcdef"
	ledgerSuite       = "github.com/JoeNutt/open-ci-control-plane/internal/change/rules"
	ledgerAlwaysRun   = "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	ledgerChangedFile = "internal/change/rules/rules.go"
	ledgerTest        = "TestRulesClassify"
)

func TestADispatchedPlanSurvivesARestart(t *testing.T) {
	path := ledgerPath(t)
	dispatched := dispatchedPlanOf(t)

	if err := jsonl.NewPlanLedger(openedLog(t, path)).RecordDispatch(context.Background(), dispatched); err != nil {
		t.Fatalf("record dispatch: %v", err)
	}

	found, err := jsonl.NewPlanLedger(openedLog(t, path)).DispatchedPlanFor(
		context.Background(), dispatched.Locator.GovernedRepository(), dispatched.Head)
	if err != nil {
		t.Fatalf("after a restart the dispatched plan is not found: %v", err)
	}
	if found.Locator != dispatched.Locator || found.Head != dispatched.Head ||
		found.DecisionID != dispatched.DecisionID || !slices.Equal(found.Suites, dispatched.Suites) ||
		!slices.Equal(found.AlwaysRun, dispatched.AlwaysRun) || !slices.Equal(found.ChangedFiles, dispatched.ChangedFiles) ||
		found.Recorded.IsRecorded != dispatched.Recorded.IsRecorded || found.Modes != dispatched.Modes ||
		!reflect.DeepEqual(found.Recorded, dispatched.Recorded) {
		t.Errorf("after a restart the ledger holds %+v, want %+v", found, dispatched)
	}
}

func ledgerPath(t *testing.T) string {
	t.Helper()

	return filepath.Join(t.TempDir(), "plan-ledger.jsonl")
}

func openedLog(t *testing.T, path string) *jsonl.Log {
	t.Helper()

	history, err := jsonl.OpenLog(path, domain.DeclaredHistoryLimits(), func() time.Time { return recordedAt })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return history
}

func wholePackageOf(t *testing.T, suite domain.CoverageSuite) []domain.PackageTests {
	t.Helper()

	pkg, err := domain.ParseGoImportPath(suite.String())
	if err != nil {
		t.Fatalf("parse import path: %v", err)
	}
	return []domain.PackageTests{domain.NewPackageTests(pkg, nil)}
}

func namedEntryOf(t *testing.T, suite domain.CoverageSuite) []domain.PackageTests {
	t.Helper()

	name, err := domain.ParseTopLevelTestName(ledgerTest)
	if err != nil {
		t.Fatalf("parse test name: %v", err)
	}
	return []domain.PackageTests{domain.NewPackageTests(wholePackageOf(t, suite)[0].Package(), []domain.TopLevelTestName{name})}
}

func dispatchedPlanOf(t *testing.T) domain.DispatchedPlan {
	t.Helper()

	locator, err := domain.ParsePullRequestLocator(ledgerRepository, 7)
	if err != nil {
		t.Fatalf("parse locator: %v", err)
	}
	head, err := domain.ParseCommitSHA(ledgerHeadSHA)
	if err != nil {
		t.Fatalf("parse head: %v", err)
	}
	id, err := domain.ParseDecisionID(ledgerDecisionID)
	if err != nil {
		t.Fatalf("parse decision ID: %v", err)
	}
	suite, err := domain.ParseCoverageSuite(ledgerSuite)
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	alwaysRun, err := domain.ParseGoImportPath(ledgerAlwaysRun)
	if err != nil {
		t.Fatalf("parse always-run suite: %v", err)
	}
	changed, err := domain.ParseRepositoryPath(ledgerChangedFile)
	if err != nil {
		t.Fatalf("parse changed file: %v", err)
	}
	return domain.DispatchedPlan{
		Locator: locator, Head: head, DecisionID: id, Suites: []domain.CoverageSuite{suite},
		AlwaysRun:    []domain.GoImportPath{alwaysRun},
		Recorded:     domain.RecordedSets{IsRecorded: true, Selected: namedEntryOf(t, suite), Minimised: wholePackageOf(t, suite)},
		ChangedFiles: []domain.RepositoryPath{changed},
		Modes:        domain.PlanModes{Selection: domain.TestModeShadow, Minimisation: domain.TestModeShadow},
	}
}

func TestAnotherRepositorysPlanAtTheSameHeadIsNotFound(t *testing.T) {
	ledger := jsonl.NewPlanLedger(openedLog(t, ledgerPath(t)))
	dispatched := dispatchedPlanOf(t)
	if err := ledger.RecordDispatch(context.Background(), dispatched); err != nil {
		t.Fatalf("record dispatch: %v", err)
	}
	stranger, err := domain.ParseGovernedRepository("JoeNutt/another-repository")
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}

	_, err = ledger.DispatchedPlanFor(context.Background(), stranger, dispatched.Head)

	if !errors.Is(err, domain.ErrNoPlanForHead) {
		t.Errorf("another repository's lookup at %s gave %v, want %v", dispatched.Head, err, domain.ErrNoPlanForHead)
	}
}

func TestASecondReportForOneHeadIsRefusedEvenAfterARestart(t *testing.T) {
	path := ledgerPath(t)
	dispatched := dispatchedPlanOf(t)
	repository := dispatched.Locator.GovernedRepository()
	before := jsonl.NewPlanLedger(openedLog(t, path))
	if err := before.RecordDispatch(context.Background(), dispatched); err != nil {
		t.Fatalf("record dispatch: %v", err)
	}
	if err := before.RecordReport(context.Background(), repository, dispatched.Head); err != nil {
		t.Fatalf("the first report was refused: %v", err)
	}

	err := jsonl.NewPlanLedger(openedLog(t, path)).RecordReport(context.Background(), repository, dispatched.Head)

	if !errors.Is(err, domain.ErrRunAlreadyReported) {
		t.Errorf("a second report after a restart gave %v, want %v", err, domain.ErrRunAlreadyReported)
	}
}

const concurrentReports = 32

func TestOfConcurrentReportsForOneHeadExactlyOneIsAccepted(t *testing.T) {
	dispatched := dispatchedPlanOf(t)
	ledger := jsonl.NewPlanLedger(openedLog(t, ledgerPath(t)))
	if err := ledger.RecordDispatch(context.Background(), dispatched); err != nil {
		t.Fatalf("record dispatch: %v", err)
	}
	start := make(chan struct{})
	outcomes := make(chan error, concurrentReports)
	var reporting sync.WaitGroup
	for range concurrentReports {
		reporting.Go(func() {
			<-start
			outcomes <- ledger.RecordReport(context.Background(), dispatched.Locator.GovernedRepository(), dispatched.Head)
		})
	}

	close(start)
	reporting.Wait()
	close(outcomes)

	accepted := 0
	for err := range outcomes {
		if err == nil {
			accepted++
		} else if !errors.Is(err, domain.ErrRunAlreadyReported) {
			t.Errorf("a concurrent report gave %v, want nil or %v", err, domain.ErrRunAlreadyReported)
		}
	}
	if accepted != 1 {
		t.Errorf("%d of %d concurrent reports for one head were accepted, want 1", accepted, concurrentReports)
	}
}

func TestADispatchedLineThatDoesNotParseIsRefusedNeverReturned(t *testing.T) {
	history := openedLog(t, ledgerPath(t))
	hostile := `{"kind":"dispatched","repository":"` + ledgerRepository + `","pullRequest":7,"head":"` +
		ledgerHeadSHA + `","decisionId":"dec_not-a-decision","suites":[]}`
	if err := history.Append(json.RawMessage(hostile)); err != nil {
		t.Fatalf("append: %v", err)
	}
	head, err := domain.ParseCommitSHA(ledgerHeadSHA)
	if err != nil {
		t.Fatalf("parse head: %v", err)
	}
	repository, err := domain.ParseGovernedRepository(ledgerRepository)
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}

	_, err = jsonl.NewPlanLedger(history).DispatchedPlanFor(context.Background(), repository, head)

	if !errors.Is(err, domain.ErrMalformedDecisionID) {
		t.Errorf("a ledger line with a malformed decision ID gave %v, want %v", err, domain.ErrMalformedDecisionID)
	}
}

func TestTheLatestPlanDispatchedForAHeadIsTheOneFound(t *testing.T) {
	ledger := jsonl.NewPlanLedger(openedLog(t, ledgerPath(t)))
	earlier := dispatchedPlanOf(t)
	later := earlier
	later.Suites = nil
	id, err := domain.ParseDecisionID("dec_ffffffffffffffffffffffffffffffff")
	if err != nil {
		t.Fatalf("parse decision ID: %v", err)
	}
	later.DecisionID = id
	for _, dispatched := range []domain.DispatchedPlan{earlier, later} {
		if err := ledger.RecordDispatch(context.Background(), dispatched); err != nil {
			t.Fatalf("record dispatch: %v", err)
		}
	}

	found, err := ledger.DispatchedPlanFor(context.Background(), earlier.Locator.GovernedRepository(), earlier.Head)

	if err != nil || found.DecisionID != later.DecisionID {
		t.Errorf("a head dispatched twice gave the plan under %s (%v), want the later one under %s",
			found.DecisionID, err, later.DecisionID)
	}
}

func TestTheLedgerListsOneRunPerHeadSayingWhetherItsResultsArrived(t *testing.T) {
	ledger := jsonl.NewPlanLedger(openedLog(t, ledgerPath(t)))
	reported := dispatchedPlanOf(t)
	unreported := dispatchedPlanOf(t)
	otherHead, err := domain.ParseCommitSHA("0a1b2c3d4e5f60718293a4b5c6d7e8f901234567")
	if err != nil {
		t.Fatalf("parse head: %v", err)
	}
	unreported.Head = otherHead
	for _, dispatched := range []domain.DispatchedPlan{reported, reported, unreported} {
		if err := ledger.RecordDispatch(context.Background(), dispatched); err != nil {
			t.Fatalf("record dispatch: %v", err)
		}
	}
	if err := ledger.RecordReport(context.Background(), reported.Locator.GovernedRepository(), reported.Head); err != nil {
		t.Fatalf("record report: %v", err)
	}

	runs, err := ledger.ShadowedRuns(context.Background())

	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	isReportedByHead := map[domain.CommitSHA]bool{}
	for _, run := range runs {
		isReportedByHead[run.Plan.Head] = run.IsReported
	}
	if len(runs) != 2 || !isReportedByHead[reported.Head] || isReportedByHead[unreported.Head] {
		t.Errorf("the ledger lists %d runs, reported by head %v; want 2, %s reported and %s not",
			len(runs), isReportedByHead, reported.Head, unreported.Head)
	}
}

func TestAShadowPlanWhoseResultsNeverArrivedIsCountedAGapFromTheLedger(t *testing.T) {
	ledger := jsonl.NewPlanLedger(openedLog(t, ledgerPath(t)))
	if err := ledger.RecordDispatch(context.Background(), dispatchedPlanOf(t)); err != nil {
		t.Fatalf("record dispatch: %v", err)
	}
	runs, err := ledger.ShadowedRuns(context.Background())
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}

	window := domain.ShadowWindowOf(domain.ShadowedSelection, runs, nil)

	if window.Runs != 1 || window.Gaps != 1 || window.Escapes != 0 {
		t.Errorf("a shadow run with no results counts %+v, want one run that is a gap", window)
	}
}

func TestALedgerRunCarriesTheTimeOfItsLatestDispatch(t *testing.T) {
	ledger := jsonl.NewPlanLedger(openedLog(t, ledgerPath(t)))
	if err := ledger.RecordDispatch(context.Background(), dispatchedPlanOf(t)); err != nil {
		t.Fatalf("record dispatch: %v", err)
	}

	runs, err := ledger.ShadowedRuns(context.Background())

	if err != nil || len(runs) != 1 || !runs[0].At.Equal(recordedAt) {
		t.Errorf("the run carries %v (%v), want the dispatch's time %v", runs, err, recordedAt)
	}
}
