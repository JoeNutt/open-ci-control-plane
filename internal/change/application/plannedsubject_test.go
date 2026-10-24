// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/extract/golang"
)

func TestAPassingChangeIsPlannedWithTheTestsItsChangedTestFileEdits(t *testing.T) {
	locator := gatedLocator(t)
	stated := ownerTestModified()
	planner := &recordingPlanner{}
	useCase := passingGateChange(t, &publishedCheckRuns{}, &dispatchedPlans{})
	useCase.Build = evidenceBuildOver(t, locator,
		recordedAtBothEnds(t, locator, stated.File.Path, ownerTestAtBase, ownerTestAtHead), golang.Extractor{}, stated)
	useCase.Planner = planner

	if err := useCase.Execute(context.Background(), locator, decisionIDOf(t, gatedDecisionID)); err != nil {
		t.Fatalf("gate change: %v", err)
	}

	if len(planner.planned) != 1 {
		t.Fatalf("planned %d subjects, want 1", len(planner.planned))
	}
	if want := evidenceOfOwnerTest(t, stated); !reflect.DeepEqual(planner.planned[0].Evidence, want) {
		t.Fatalf("planned with evidence %+v, want %+v", planner.planned[0].Evidence, want)
	}
}

func evidenceOfOwnerTest(t *testing.T, stated domain.StatedFile) domain.ChangeEvidence {
	t.Helper()

	changed, err := domain.ParseChangedFile(stated)
	if err != nil {
		t.Fatalf("parse changed file: %v", err)
	}
	binding, err := bindingForEveryRepository{}.BindingOf(context.Background(), gatedLocator(t).GovernedRepository())
	if err != nil {
		t.Fatalf("binding: %v", err)
	}
	evidence, err := domain.ParseChangeEvidence([]domain.ChangedFile{changed}, editedTestsAtHead(t, stated, ownerTestAtHead), binding)
	if err != nil {
		t.Fatalf("parse change evidence: %v", err)
	}
	return evidence
}

type recordingPlanner struct {
	planned []domain.PlanSubject
}

func (planner *recordingPlanner) Execute(_ context.Context, subject domain.PlanSubject) (domain.TestPlan, error) {
	planner.planned = append(planner.planned, subject)
	return domain.FailedPlanFor(subject)
}
