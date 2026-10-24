// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
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

func hourlyCleanHistoryOf(t *testing.T, repository string) []domain.ShadowedRun {
	t.Helper()

	inShadow := domain.DispatchedPlan{
		Locator:  gatedLocatorIn(t, repository),
		Modes:    domain.PlanModes{Selection: domain.TestModeShadow, Minimisation: domain.TestModeShadow},
		Recorded: domain.RecordedSets{IsRecorded: true},
	}
	var runs []domain.ShadowedRun
	for at := gatedAt.AddDate(0, -2, 0); at.Before(gatedAt); at = at.Add(time.Hour) {
		runs = append(runs, domain.ShadowedRun{At: at, Plan: inShadow, IsReported: true})
	}
	return runs
}

func anEscapeAnHourAgo(t *testing.T, mode domain.ShadowedMode, repository string) []domain.Escape {
	t.Helper()

	return []domain.Escape{{Mode: mode, PullRequest: gatedLocatorIn(t, repository), At: gatedAt.Add(-time.Hour)}}
}

func TestAnEnforcedSelectionWithNoShadowHistoryDropsToShadowNamingWhy(t *testing.T) {
	plan := modedPlanOf(t, "enforce", "off").GatedOn(nil, nil, gatedAt)

	if !plan.FullMatrix || len(plan.Tests) != 0 || plan.Trigger != domain.PlanTriggerShadowUnproven ||
		plan.Modes.Selection != domain.TestModeShadow || !plan.Recorded.IsRecorded || len(plan.Recorded.Selected) != 2 {
		t.Errorf("an unproven enforced plan is %+v, want the full matrix in shadow with %q and both selected suites recorded",
			plan, domain.PlanTriggerShadowUnproven)
	}
}

func TestAnEnforcedSelectionWithALongCleanShadowHistoryRunsItsSelection(t *testing.T) {
	plan := modedPlanOf(t, "enforce", "off").GatedOn(hourlyCleanHistoryOf(t, thisRepository), nil, gatedAt)

	if plan.FullMatrix || len(plan.Tests) != 2 || plan.Modes.Selection != domain.TestModeEnforce {
		t.Errorf("a proven enforced plan is %+v, want both selected suites run under enforce", plan)
	}
}

func TestAnEscapeLatchesAnEnforcedSelectionBackToShadow(t *testing.T) {
	escapes := anEscapeAnHourAgo(t, domain.ShadowedSelection, thisRepository)

	plan := modedPlanOf(t, "enforce", "off").GatedOn(hourlyCleanHistoryOf(t, thisRepository), escapes, gatedAt)

	if !plan.FullMatrix || plan.Trigger != domain.PlanTriggerEscapeRecorded || plan.Modes.Selection != domain.TestModeShadow {
		t.Errorf("a latched plan is %+v, want the full matrix in shadow with %q", plan, domain.PlanTriggerEscapeRecorded)
	}
}

func TestAnotherRepositorysHistoryNeitherProvesNorLatchesThisOne(t *testing.T) {
	unproven := modedPlanOf(t, "enforce", "off").GatedOn(hourlyCleanHistoryOf(t, "acme/payments"), nil, gatedAt)
	unlatched := modedPlanOf(t, "enforce", "off").GatedOn(
		hourlyCleanHistoryOf(t, thisRepository), anEscapeAnHourAgo(t, domain.ShadowedSelection, "acme/payments"), gatedAt)

	if unproven.Trigger != domain.PlanTriggerShadowUnproven {
		t.Errorf("another repository's clean history gave %q, want %q", unproven.Trigger, domain.PlanTriggerShadowUnproven)
	}
	if unlatched.Modes.Selection != domain.TestModeEnforce {
		t.Errorf("another repository's escape latched this one: %+v", unlatched)
	}
}

func TestWhenSelectionDropsToShadowAnEnforcedMinimisationDropsWithItAndRecordsItsSet(t *testing.T) {
	escapes := anEscapeAnHourAgo(t, domain.ShadowedSelection, thisRepository)

	plan := modedPlanOf(t, "enforce", "enforce").GatedOn(hourlyCleanHistoryOf(t, thisRepository), escapes, gatedAt)

	if plan.Modes.Minimisation != domain.TestModeShadow || len(plan.Recorded.Minimised) != 2 {
		t.Errorf("minimisation is %s with minimised %v, want shadow with both suites recorded", plan.Modes.Minimisation, plan.Recorded.Minimised)
	}
}

func TestAnUnprovenMinimisationUnderAnEnforcedSelectionShadowsAloneAndRecordsItsSet(t *testing.T) {
	escapes := anEscapeAnHourAgo(t, domain.ShadowedMinimisation, thisRepository)

	plan := modedPlanOf(t, "enforce", "enforce").GatedOn(hourlyCleanHistoryOf(t, thisRepository), escapes, gatedAt)

	if plan.FullMatrix || len(plan.Tests) != 2 || plan.Modes.Selection != domain.TestModeEnforce ||
		plan.Modes.Minimisation != domain.TestModeShadow || len(plan.Recorded.Minimised) != 2 {
		t.Errorf("the plan is %+v, want the selection run, minimisation in shadow with both suites recorded as minimised", plan)
	}
}
