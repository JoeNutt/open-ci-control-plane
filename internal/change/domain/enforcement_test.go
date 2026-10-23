// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	day               = 24 * time.Hour
	shadowWindow      = 30 * day
	minimumShadowRuns = 20
)

var gatedAt = time.Date(2026, 11, 30, 12, 0, 0, 0, time.UTC)

func gatedLocatorIn(t *testing.T, repository string) domain.PullRequestLocator {
	t.Helper()

	locator, err := domain.ParsePullRequestLocator(repository, 7)
	if err != nil {
		t.Fatalf("parse locator: %v", err)
	}
	return locator
}

func shadowRunAt(t *testing.T, at time.Time, isRecorded bool) domain.ShadowedRun {
	t.Helper()

	inShadow := domain.PlanModes{Selection: domain.TestModeShadow, Minimisation: domain.TestModeShadow}
	return domain.ShadowedRun{
		At: at,
		Plan: domain.DispatchedPlan{
			Locator: gatedLocatorIn(t, thisRepository), Modes: inShadow, Recorded: domain.RecordedSets{IsRecorded: isRecorded},
		},
		IsReported: true,
	}
}

func evidenceRuns(t *testing.T, count int, oldest time.Time) []domain.ShadowedRun {
	t.Helper()

	runs := make([]domain.ShadowedRun, 0, count)
	for index := range count {
		runs = append(runs, shadowRunAt(t, oldest.Add(time.Duration(index)*time.Hour), true))
	}
	return runs
}

func provenRuns(t *testing.T) []domain.ShadowedRun {
	t.Helper()

	return append(evidenceRuns(t, minimumShadowRuns, gatedAt.Add(-10*day)), shadowRunAt(t, gatedAt.Add(-shadowWindow-day), false))
}

func escapeOf(t *testing.T, mode domain.ShadowedMode, at time.Time) []domain.Escape {
	t.Helper()

	return []domain.Escape{{Mode: mode, PullRequest: gatedLocatorIn(t, thisRepository), At: at}}
}

func selectionGateOn(t *testing.T, runs []domain.ShadowedRun, escapes []domain.Escape) domain.PlanTrigger {
	t.Helper()

	plan := modedPlanOf(t, "enforce", "off").GatedOn(runs, escapes, gatedAt)
	if plan.Modes.Selection == domain.TestModeEnforce {
		return domain.PlanTriggerNone
	}
	return plan.Trigger
}

func TestASelectionWithAnEscapeInTheLast30DaysIsNotEnforcedAndNamesEscapeRecorded(t *testing.T) {
	escapes := escapeOf(t, domain.ShadowedSelection, gatedAt.Add(-shadowWindow+day))

	if trigger := selectionGateOn(t, provenRuns(t), escapes); trigger != domain.PlanTriggerEscapeRecorded {
		t.Errorf("a selection with an escape 29 days ago gave %q, want %q", trigger, domain.PlanTriggerEscapeRecorded)
	}
}

func TestASelectionWhoseShadowHistoryIsYoungerThan30DaysIsNotEnforced(t *testing.T) {
	runs := evidenceRuns(t, minimumShadowRuns+5, gatedAt.Add(-shadowWindow+day))

	if trigger := selectionGateOn(t, runs, nil); trigger != domain.PlanTriggerShadowUnproven {
		t.Errorf("29 days of clean shadow history gave %q, want %q", trigger, domain.PlanTriggerShadowUnproven)
	}
}

func TestASelectionIsEnforcedOnlyWithQ1sMinimumOfEvidenceRunsInTheWindow(t *testing.T) {
	old := shadowRunAt(t, gatedAt.Add(-shadowWindow-day), true)
	recent := gatedAt.Add(-10 * day)
	gap := shadowRunAt(t, recent, false)
	for name, bounded := range map[string]struct {
		runs []domain.ShadowedRun
		want domain.PlanTrigger
	}{
		"one under the minimum": {append(evidenceRuns(t, minimumShadowRuns-1, recent), old), domain.PlanTriggerShadowUnproven},
		"at the minimum":        {append(evidenceRuns(t, minimumShadowRuns, recent), old), domain.PlanTriggerNone},
		"gaps make up the rest": {append(append(evidenceRuns(t, minimumShadowRuns-1, recent), gap), old), domain.PlanTriggerShadowUnproven},
	} {
		if trigger := selectionGateOn(t, bounded.runs, nil); trigger != bounded.want {
			t.Errorf("%s: gave %q, want %q", name, trigger, bounded.want)
		}
	}
}

func TestTheLatchHoldsOnlyForThisModesEscapesInsideTheWindow(t *testing.T) {
	for name, escapes := range map[string][]domain.Escape{
		"an escape older than the window":  escapeOf(t, domain.ShadowedSelection, gatedAt.Add(-shadowWindow-time.Hour)),
		"an escape against the other mode": escapeOf(t, domain.ShadowedMinimisation, gatedAt.Add(-time.Hour)),
	} {
		if trigger := selectionGateOn(t, provenRuns(t), escapes); trigger != domain.PlanTriggerNone {
			t.Errorf("%s latched selection with %q", name, trigger)
		}
	}
}

func TestAnotherRepositorysEvidenceAndEscapesDoNotGateThisOne(t *testing.T) {
	elsewhere := gatedLocatorIn(t, "acme/payments")
	runs := provenRuns(t)
	for index := range runs {
		runs[index].Plan.Locator = elsewhere
	}
	escapes := escapeOf(t, domain.ShadowedSelection, gatedAt.Add(-time.Hour))
	escapes[0].PullRequest = elsewhere

	if trigger := selectionGateOn(t, runs, nil); trigger != domain.PlanTriggerShadowUnproven {
		t.Errorf("another repository's proven runs gave %q, want %q", trigger, domain.PlanTriggerShadowUnproven)
	}
	if trigger := selectionGateOn(t, provenRuns(t), escapes); trigger != domain.PlanTriggerNone {
		t.Errorf("another repository's escape latched this one with %q", trigger)
	}
}

func TestAnEnforcedPlanWhoseSelectionIsUnprovenDropsToShadowNamingWhy(t *testing.T) {
	plan := modedPlanOf(t, "enforce", "off").GatedOn(nil, nil, gatedAt)

	if !plan.FullMatrix || len(plan.Suites) != 0 || plan.Trigger != domain.PlanTriggerShadowUnproven ||
		plan.Modes.Selection != domain.TestModeShadow || !plan.Recorded.IsRecorded || len(plan.Recorded.Selected) != 2 {
		t.Errorf("an unproven enforced plan is %+v, want the full matrix in shadow with %q and both selected suites recorded",
			plan, domain.PlanTriggerShadowUnproven)
	}
}

func TestWhenSelectionDropsToShadowAnEnforcedMinimisationDropsWithItAndRecordsItsSet(t *testing.T) {
	escapes := escapeOf(t, domain.ShadowedSelection, gatedAt.Add(-time.Hour))

	plan := modedPlanOf(t, "enforce", "enforce").GatedOn(provenRuns(t), escapes, gatedAt)

	if plan.Modes.Minimisation != domain.TestModeShadow || len(plan.Recorded.Minimised) != 2 {
		t.Errorf("minimisation is %s with minimised %v, want shadow with both suites recorded", plan.Modes.Minimisation, plan.Recorded.Minimised)
	}
}

func TestAnUnprovenMinimisationUnderAnEnforcedSelectionShadowsAloneAndRecordsItsSet(t *testing.T) {
	escapes := escapeOf(t, domain.ShadowedMinimisation, gatedAt.Add(-time.Hour))

	plan := modedPlanOf(t, "enforce", "enforce").GatedOn(provenRuns(t), escapes, gatedAt)

	if plan.FullMatrix || len(plan.Suites) != 2 || plan.Modes.Selection != domain.TestModeEnforce ||
		plan.Modes.Minimisation != domain.TestModeShadow || len(plan.Recorded.Minimised) != 2 {
		t.Errorf("the plan is %+v, want the selection run, minimisation in shadow with both suites recorded as minimised", plan)
	}
}
