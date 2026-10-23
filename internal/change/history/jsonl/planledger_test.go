// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl_test

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/history/jsonl"
)

const (
	ledgerRepository = "JoeNutt/open-ci-control-plane"
	ledgerHeadSHA    = "9f1cabc0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6"
	ledgerDecisionID = "dec_0123456789abcdef0123456789abcdef"
	ledgerSuite      = "github.com/JoeNutt/open-ci-control-plane/internal/change/rules"
)

func TestADispatchedPlanSurvivesARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan-ledger.jsonl")
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
		found.DecisionID != dispatched.DecisionID || !slices.Equal(found.Suites, dispatched.Suites) {
		t.Errorf("after a restart the ledger holds %+v, want %+v", found, dispatched)
	}
}

func openedLog(t *testing.T, path string) *jsonl.Log {
	t.Helper()

	history, err := jsonl.OpenLog(path, domain.DeclaredHistoryLimits(), func() time.Time { return recordedAt })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return history
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
	return domain.DispatchedPlan{Locator: locator, Head: head, DecisionID: id, Suites: []domain.CoverageSuite{suite}}
}
