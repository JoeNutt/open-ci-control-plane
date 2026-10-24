// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const uncataloguedPolicyData = `{"catalog": {}, "bindings": {"acme/payments": {"policies": [{
	"use": "mvvm", "import_roots": [], "paths": {}
}]}}}`

func TestAChangeThatDoesNotPassIsNeverDispatched(t *testing.T) {
	cases := map[string]func(t *testing.T) (domain.FamilyEngine, application.BuildDocument){
		"a structural refusal": func(t *testing.T) (domain.FamilyEngine, application.BuildDocument) {
			return answeringEngine{decided: blockedDecision(t)}, buildOverAnAuthChange(t, gatedLocator(t))
		},
		"an undefined decision": func(t *testing.T) (domain.FamilyEngine, application.BuildDocument) {
			return answeringEngine{decided: domain.UndecidedDecision(revisionForTest(t))}, buildOverAnAuthChange(t, gatedLocator(t))
		},
		"a policy fault": func(t *testing.T) (domain.FamilyEngine, application.BuildDocument) {
			return faultingEngine{fault: errors.New("conflicting rule outputs")}, buildOverAnAuthChange(t, gatedLocator(t))
		},
		"no binding": func(t *testing.T) (domain.FamilyEngine, application.BuildDocument) {
			policyData := `{"catalog": {` + mvvmCatalog + `}, "bindings": {` + otherRepositoryBinding + `}}`
			return structuralEngineServing(t, policyData), buildOverAnAuthChange(t, gatedLocator(t))
		},
		"an unknown policy": func(t *testing.T) (domain.FamilyEngine, application.BuildDocument) {
			return structuralEngineServing(t, uncataloguedPolicyData), buildOverAnAuthChange(t, gatedLocator(t))
		},
		"an over-bound change": func(t *testing.T) (domain.FamilyEngine, application.BuildDocument) {
			return answeringEngine{decided: allowedDecision(t)}, buildOverRecorded(t, gatedLocator(t), domain.File{
				Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
				PatchBytes: domain.DeclaredLimits().MaxPatchBytesPerFile + 1,
			})
		},
	}

	for form, arrange := range cases {
		t.Run(form, func(t *testing.T) {
			engine, build := arrange(t)
			dispatched := &dispatchedPlans{}
			useCase := passingGateChange(t, &publishedCheckRuns{}, dispatched)
			useCase.Gate = gateOver(t, engine)
			useCase.Build = build

			_ = useCase.Execute(context.Background(), gatedLocator(t), decisionIDOf(t, gatedDecisionID))

			if plans := dispatched.Dispatched(); len(plans) != 0 {
				t.Errorf("dispatched %d plans for a change that did not pass", len(plans))
			}
		})
	}
}

func TestAChangeThatDoesNotPassNeverReadsItsChangeEvidence(t *testing.T) {
	locator := gatedLocator(t)
	stated := ownerTestModified()
	contents := &countedFiles{recorded: recordedAt(t, locator, recordedHeadSHA, stated.File.Path, ownerTestAtHead)}
	useCase := passingGateChange(t, &publishedCheckRuns{}, &dispatchedPlans{})
	useCase.Gate = gateOver(t, answeringEngine{decided: blockedDecision(t)})
	useCase.Build = evidenceBuildOver(t, locator, contents, claimsNothing{}, stated)

	if err := useCase.Execute(context.Background(), locator, decisionIDOf(t, gatedDecisionID)); err != nil {
		t.Fatalf("gate change: %v", err)
	}

	if contents.asked != 0 {
		t.Fatalf("the forge was asked for %d file contents for a change that plans nothing", contents.asked)
	}
}

func TestADispatchFailureLeavesThePostedVerdictAndIsReported(t *testing.T) {
	published := &publishedCheckRuns{}
	useCase := passingGateChange(t, published, &dispatchedPlans{})
	useCase.Dispatcher = refusingDispatcher{}

	err := useCase.Execute(context.Background(), gatedLocator(t), decisionIDOf(t, gatedDecisionID))

	if !errors.Is(err, errDispatchRefused) {
		t.Errorf("error is %v, want %v", err, errDispatchRefused)
	}
	run := published.PublishedOn(gatedLocator(t), commitSHAOf(t, recordedHeadSHA))[structuralCheckRunName]
	if run.Conclusion != domain.ConclusionSuccess {
		t.Errorf("the posted verdict is %q after a failed dispatch, want %q", run.Conclusion, domain.ConclusionSuccess)
	}
}

var errDispatchRefused = errors.New("the forge refused the dispatch")

type refusingDispatcher struct{}

func (refusingDispatcher) DispatchPlan(context.Context, domain.TestPlan) error {
	return errDispatchRefused
}

func gateOver(t *testing.T, engine domain.FamilyEngine) domain.Gate {
	t.Helper()

	gate, err := domain.ParseGate(map[domain.PolicyFamily]domain.FamilyEngine{domain.PolicyFamilyStructural: engine})
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}
	return gate
}

func blockedDecision(t *testing.T) domain.Decision {
	t.Helper()

	decided, err := domain.ParseDecision([]byte(`{
		"input_wellformed": true, "schema_version_accepted": 1, "effect": "block",
		"violations": [{"rule_id": "clean-architecture", "message": "refused", "severity": "deny"}],
		"reviewers": [], "fail_closed": true
	}`), revisionForTest(t))
	if err != nil {
		t.Fatalf("parse decision: %v", err)
	}
	return decided
}
