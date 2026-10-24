// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAPassingChangeIsPlannedWithTheTestsItsChangedTestFileEdits(t *testing.T) {
	locator := gatedLocator(t)
	stated := ownerTestModified()
	build := evidenceBuildOver(t, locator,
		recordedAtBothEnds(t, locator, stated.File.Path, ownerTestAtBase, ownerTestAtHead), importsNothing{}, stated)

	subject := plannedSubjectOf(t, build)

	if want := evidenceOf(t, editedTestsAtHead(t, stated, ownerTestAtHead), stated); !reflect.DeepEqual(subject.Evidence, want) {
		t.Fatalf("planned with evidence %+v, want %+v", subject.Evidence, want)
	}
}

type recordingPlanner struct {
	planned []domain.PlanSubject
}

func (planner *recordingPlanner) Execute(_ context.Context, subject domain.PlanSubject) (domain.TestPlan, error) {
	planner.planned = append(planner.planned, subject)
	return domain.FailedPlanFor(subject)
}
