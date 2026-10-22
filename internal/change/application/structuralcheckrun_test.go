// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

const noPolicyLoaded = "no policy loaded"

func TestAnUndefinedStructuralDecisionFailsReadingNoPolicyLoaded(t *testing.T) {
	run := structuralCheckRunAfter(t, answeringEngine{decided: domain.UndecidedDecision(revisionForTest(t))})

	if run.Conclusion != domain.ConclusionFailure {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
	}
	if !strings.Contains(strings.ToLower(run.Title), noPolicyLoaded) {
		t.Errorf("title %q does not read %q", run.Title, noPolicyLoaded)
	}
}

func TestAStructuralPolicyDeclaringItselfOpenStillFailsWhenUndefined(t *testing.T) {
	undecided, err := domain.ParseDecision([]byte(`{
		"input_wellformed": false, "schema_version_accepted": 1, "effect": "allow",
		"violations": [], "reviewers": [], "fail_closed": false
	}`), revisionForTest(t))
	if err != nil {
		t.Fatalf("parse decision: %v", err)
	}

	run := structuralCheckRunAfter(t, answeringEngine{decided: undecided})

	if run.Conclusion != domain.ConclusionFailure {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
	}
}

const policyFault = "policy fault"

func TestAnEvaluationErrorFailsReportingAPolicyFaultAndNoRule(t *testing.T) {
	run := structuralCheckRunAfter(t, faultingEngine{fault: errors.New("conflicting rule outputs")})

	if run.Conclusion != domain.ConclusionFailure {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
	}
	if !strings.Contains(strings.ToLower(run.Title), policyFault) {
		t.Errorf("title %q does not report a %s", run.Title, policyFault)
	}
	if run.ViolationCount != 0 || len(run.Violations) != 0 {
		t.Errorf("a policy fault names rules: %+v", run.Violations)
	}
}

type faultingEngine struct {
	fault error
}

func (engine faultingEngine) Decide(context.Context, domain.Document) (domain.Decision, error) {
	return domain.Decision{}, engine.fault
}

func structuralCheckRunAfter(t *testing.T, engine domain.FamilyEngine) domain.CheckRun {
	t.Helper()

	locator := locatorOf(t, "acme/payments", 412)
	gate, err := domain.ParseGate(map[domain.PolicyFamily]domain.FamilyEngine{
		domain.PolicyFamilyStructural: engine,
	})
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}
	published := &memory.CheckRuns{}
	useCase := application.GateChange{
		Build:     buildOverAnAuthChange(t, locator),
		Gate:      gate,
		Publisher: published,
		Recorder:  &recordedDecisions{},
	}

	if err := useCase.Execute(context.Background(), locator, decisionIDOf(t, gatedDecisionID)); err != nil {
		t.Fatalf("gate change: %v", err)
	}
	runs := published.PublishedOn(locator, commitSHAOf(t, recordedHeadSHA))
	if len(runs) != 1 {
		t.Fatalf("published %d check runs, want one: %v", len(runs), runs)
	}
	run, present := runs["policy/"+string(domain.PolicyFamilyStructural)]
	if !present {
		t.Fatalf("no structural check run among %v", runs)
	}
	return run
}
