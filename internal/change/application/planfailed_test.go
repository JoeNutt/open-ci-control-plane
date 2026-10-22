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
	if !plans[0].FullMatrix || plans[0].Trigger != domain.PlanTriggerPlanFailed || len(plans[0].Suites) != 0 {
		t.Errorf("dispatched full matrix %t, trigger %q, %d suites; want the full matrix with %q",
			plans[0].FullMatrix, plans[0].Trigger, len(plans[0].Suites), domain.PlanTriggerPlanFailed)
	}
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
	useCase.Planner.History = refusingHistory{}

	err := useCase.Execute(context.Background(), gatedLocator(t), decisionIDOf(t, gatedDecisionID))

	if !errors.Is(err, domain.ErrHistoryFull) {
		t.Errorf("the store's refusal was not returned for the log: %v", err)
	}
	plans := dispatched.Dispatched()
	if len(plans) != 1 || !plans[0].FullMatrix || plans[0].Trigger != domain.PlanTriggerPlanFailed {
		t.Errorf("a refusing history store dispatched %+v, want the full matrix with %q", plans, domain.PlanTriggerPlanFailed)
	}
}
