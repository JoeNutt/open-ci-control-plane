// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAPassingChangeIsPlannedAndDispatchedOnceAfterItsCheckRun(t *testing.T) {
	locator := gatedLocator(t)
	dispatched := &dispatchedPlans{}
	published := &publishedCheckRuns{}
	useCase := passingGateChange(t, published, dispatched)

	if err := useCase.Execute(context.Background(), locator, decisionIDOf(t, gatedDecisionID)); err != nil {
		t.Fatalf("gate change: %v", err)
	}

	plans := dispatched.Dispatched()
	if len(plans) != 1 {
		t.Fatalf("dispatched %d plans, want 1", len(plans))
	}
	if plans[0].Head.String() != recordedHeadSHA || plans[0].DecisionID != decisionIDOf(t, gatedDecisionID) {
		t.Errorf("dispatched a plan for %s under %s, want %s under %s",
			plans[0].Head, plans[0].DecisionID, recordedHeadSHA, gatedDecisionID)
	}
	if _, isPosted := published.PublishedOn(locator, commitSHAOf(t, recordedHeadSHA))[structuralCheckRunName]; !isPosted {
		t.Errorf("no structural check run was posted")
	}
}

func passingGateChange(t *testing.T, published *publishedCheckRuns, dispatched *dispatchedPlans) application.GateChange {
	t.Helper()

	gate, err := domain.ParseGate(map[domain.PolicyFamily]domain.FamilyEngine{
		domain.PolicyFamilyStructural: answeringEngine{decided: allowedDecision(t)},
	})
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}
	return application.GateChange{
		Build:      buildOverAnAuthChange(t, gatedLocator(t)),
		Gate:       gate,
		Publisher:  published,
		Recorder:   &recordedDecisions{},
		Bindings:   bindingForEveryRepository{},
		Planner:    application.PlanTests{Maps: newCoverageStore(domain.DeclaredCoverageLimits()), History: readableHistory{}},
		Dispatcher: dispatched,
		Ledger:     &plannedHeads{},
	}
}

func allowedDecision(t *testing.T) domain.Decision {
	t.Helper()

	decided, err := domain.ParseDecision([]byte(`{
		"input_wellformed": true, "schema_version_accepted": 1, "effect": "allow",
		"violations": [], "reviewers": [], "fail_closed": true
	}`), revisionForTest(t))
	if err != nil {
		t.Fatalf("parse decision: %v", err)
	}
	return decided
}

type bindingForEveryRepository struct{}

func (bindingForEveryRepository) BindingOf(_ context.Context, repository domain.GovernedRepository) (domain.Binding, error) {
	return domain.ParseBinding(domain.BindingStatement{
		Repository: repository.String(),
		Workflow:   "open-ci-tests.yml",
		AlwaysRun:  []string{"example.com/app/internal/domain"},
		Policies: []domain.BoundPolicyStatement{{
			Use:         "clean-architecture",
			ImportRoots: []domain.ImportRootStatement{{Prefix: "example.com/app", Path: ""}},
			Paths:       map[string][]string{"domain": {"internal/domain/**"}},
		}},
	})
}

func TestADispatchedPlanIsInTheLedgerForItsHeadAndPullRequest(t *testing.T) {
	ledger := &plannedHeads{}
	useCase := passingGateChange(t, &publishedCheckRuns{}, &dispatchedPlans{})
	useCase.Ledger = ledger

	if err := useCase.Execute(context.Background(), gatedLocator(t), decisionIDOf(t, gatedDecisionID)); err != nil {
		t.Fatalf("gate change: %v", err)
	}

	dispatched, err := ledger.DispatchedPlanFor(context.Background(), gatedLocator(t).GovernedRepository(), commitSHAOf(t, recordedHeadSHA))
	if err != nil {
		t.Fatalf("the dispatched plan is not in the ledger: %v", err)
	}
	if dispatched.Locator != gatedLocator(t) || dispatched.Plan.DecisionID != decisionIDOf(t, gatedDecisionID) {
		t.Errorf("the ledger holds %+v under %s, want the plan for %+v under %s",
			dispatched.Locator, dispatched.Plan.DecisionID, gatedLocator(t), gatedDecisionID)
	}
}
