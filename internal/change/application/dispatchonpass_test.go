// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	coverage "github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/memory"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

func TestAPassingChangeIsPlannedAndDispatchedOnceAfterItsCheckRun(t *testing.T) {
	locator := gatedLocator(t)
	dispatched := &memory.PlanDispatches{}
	published := &memory.CheckRuns{}
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

func passingGateChange(t *testing.T, published *memory.CheckRuns, dispatched *memory.PlanDispatches) application.GateChange {
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
		Planner:    application.PlanTests{Maps: coverage.NewCoverageMaps(domain.DeclaredCoverageLimits())},
		Dispatcher: dispatched,
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
