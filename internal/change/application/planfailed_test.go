// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAPassingChangeWhosePlanCannotBeProducedRunsTheFullMatrixNamingWhy(t *testing.T) {
	dispatched := &dispatchedPlans{}
	useCase := passingGateChange(t, &publishedCheckRuns{}, dispatched)
	useCase.Planner = application.PlanTests{Maps: unreadableCoverage{}, History: readableHistory{}}

	err := useCase.Execute(context.Background(), gatedLocator(t), decisionIDOf(t, gatedDecisionID))

	if !errors.Is(err, errCoverageUnreadable) {
		t.Errorf("the planner's cause was not returned for the log: %v", err)
	}
	plans := dispatched.Dispatched()
	if len(plans) != 1 {
		t.Fatalf("dispatched %d plans, want the full matrix once", len(plans))
	}
	if !plans[0].FullMatrix || plans[0].Trigger != domain.PlanTriggerPlanFailed || len(plans[0].Tests) != 0 {
		t.Errorf("dispatched full matrix %t, trigger %q, %d packages; want the full matrix with %q",
			plans[0].FullMatrix, plans[0].Trigger, len(plans[0].Tests), domain.PlanTriggerPlanFailed)
	}
}

func TestAPassingChangeWhoseEvidenceCannotBeReadRunsTheFullMatrixNamingWhy(t *testing.T) {
	dispatched := &dispatchedPlans{}
	published := &publishedCheckRuns{}
	useCase := passingGateChange(t, published, dispatched)
	useCase.Build = evidenceBuildOver(t, gatedLocator(t), recordedFiles{}, importsNothing{}, domain.StatedFile{
		File:  domain.File{Path: "infra/iam.tf", Action: domain.FileActionModified},
		Patch: "@@ -1,3 +1,3 @@", IsPatchStated: true,
	})

	err := useCase.Execute(context.Background(), gatedLocator(t), decisionIDOf(t, gatedDecisionID))

	if !errors.Is(err, domain.ErrMalformedHunk) {
		t.Errorf("the evidence's cause was not returned for the log: %v", err)
	}
	if _, isPosted := published.PublishedOn(gatedLocator(t), commitSHAOf(t, recordedHeadSHA))[structuralCheckRunName]; !isPosted {
		t.Errorf("an evidence failure stopped the structural check run")
	}
	plans := dispatched.Dispatched()
	if len(plans) != 1 || !plans[0].FullMatrix || plans[0].Trigger != domain.PlanTriggerPlanFailed {
		t.Errorf("unreadable evidence dispatched %+v, want the full matrix with %q", plans, domain.PlanTriggerPlanFailed)
	}
}

func TestATestFileEditedOutsideAnyTestPlansNormallyUnderABindingWithNoImportRoots(t *testing.T) {
	locator := gatedLocator(t)
	stated := domain.StatedFile{
		File:  domain.File{Path: "internal/auth/owner_test.go", Action: domain.FileActionModified},
		Patch: "@@ -3,1 +3,1 @@\n-import \"testing\"\n+import \"testing\"", IsPatchStated: true,
	}
	dispatched := &dispatchedPlans{}
	useCase := passingGateChange(t, &publishedCheckRuns{}, dispatched)
	useCase.Bindings = bindingWithNoImportRoots{}
	useCase.Build = evidenceBuildOver(t, locator,
		recordedAtBothEnds(t, locator, stated.File.Path, ownerTestAtBase, ownerTestAtHead), importsNothing{}, stated)

	err := useCase.Execute(context.Background(), locator, decisionIDOf(t, gatedDecisionID))

	if err != nil {
		t.Fatalf("gate change: %v", err)
	}
	if plans := dispatched.Dispatched(); len(plans) != 1 || plans[0].Trigger == domain.PlanTriggerPlanFailed {
		t.Fatalf("dispatched %+v, want one plan that did not fail", plans)
	}
}

type bindingWithNoImportRoots struct{}

func (bindingWithNoImportRoots) BindingOf(_ context.Context, repository domain.GovernedRepository) (domain.Binding, error) {
	return domain.ParseBinding(domain.BindingStatement{
		Repository: repository.String(),
		Workflow:   "open-ci-tests.yml",
		Policies:   []domain.BoundPolicyStatement{{Use: "clean-architecture"}},
	})
}

var errBindingUnreadable = errors.New("the policy binding cannot be read")

type unreadableBinding struct{}

func (unreadableBinding) BindingOf(context.Context, domain.GovernedRepository) (domain.Binding, error) {
	return domain.Binding{}, errBindingUnreadable
}

func TestAPassingChangeWhoseBindingCannotBeReadDispatchesNothingAndSaysWhy(t *testing.T) {
	dispatched := &dispatchedPlans{}
	useCase := passingGateChange(t, &publishedCheckRuns{}, dispatched)
	useCase.Bindings = unreadableBinding{}

	err := useCase.Execute(context.Background(), gatedLocator(t), decisionIDOf(t, gatedDecisionID))

	if !errors.Is(err, errBindingUnreadable) {
		t.Errorf("the binding's cause was not returned for the log: %v", err)
	}
	if plans := dispatched.Dispatched(); len(plans) != 0 {
		t.Errorf("dispatched %d plans with no workflow to dispatch to", len(plans))
	}
}

type refusingHistory struct{}

func (refusingHistory) VerifyReadable(context.Context) error { return domain.ErrHistoryFull }

func TestAHistoryStoreRefusalPlansTheFullMatrixNeverSelection(t *testing.T) {
	dispatched := &dispatchedPlans{}
	useCase := passingGateChange(t, &publishedCheckRuns{}, dispatched)
	useCase.Planner = application.PlanTests{Maps: newCoverageStore(domain.DeclaredCoverageLimits()), History: refusingHistory{}}

	err := useCase.Execute(context.Background(), gatedLocator(t), decisionIDOf(t, gatedDecisionID))

	if !errors.Is(err, domain.ErrHistoryFull) {
		t.Errorf("the store's refusal was not returned for the log: %v", err)
	}
	plans := dispatched.Dispatched()
	if len(plans) != 1 || !plans[0].FullMatrix || plans[0].Trigger != domain.PlanTriggerPlanFailed {
		t.Errorf("a refusing history store dispatched %+v, want the full matrix with %q", plans, domain.PlanTriggerPlanFailed)
	}
}
