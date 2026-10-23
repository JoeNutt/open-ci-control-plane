// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var gatedAt = time.Date(2026, 11, 30, 12, 0, 0, 0, time.UTC)

func TestAModeWithAnEscapeInTheLast30DaysIsNotEnforcedAndNamesEscapeRecorded(t *testing.T) {
	escapes := []domain.Escape{{Mode: domain.ShadowedSelection, At: gatedAt.Add(-domain.ShadowWindowLength + 24*time.Hour)}}

	verdict := domain.EnforcementOf(domain.ShadowedSelection, nil, escapes, gatedAt)

	if verdict.IsEnforced || verdict.Trigger != domain.PlanTriggerEscapeRecorded {
		t.Errorf("a mode with an escape 29 days ago gave %+v, want not enforced with %q", verdict, domain.PlanTriggerEscapeRecorded)
	}
}

func evidenceRuns(count int, oldest time.Time) []domain.ShadowedRun {
	inShadow := domain.PlanModes{Selection: domain.TestModeShadow, Minimisation: domain.TestModeShadow}
	runs := make([]domain.ShadowedRun, 0, count)
	for index := range count {
		runs = append(runs, domain.ShadowedRun{
			At:         oldest.Add(time.Duration(index) * time.Hour),
			Plan:       domain.DispatchedPlan{Modes: inShadow, Recorded: domain.RecordedSets{IsRecorded: true}},
			IsReported: true,
		})
	}
	return runs
}

func TestAModeWhoseShadowHistoryIsYoungerThan30DaysIsNotEnforced(t *testing.T) {
	runs := evidenceRuns(domain.MinimumShadowRuns+5, gatedAt.Add(-domain.ShadowWindowLength+24*time.Hour))

	verdict := domain.EnforcementOf(domain.ShadowedSelection, runs, nil, gatedAt)

	if verdict.IsEnforced || verdict.Trigger != domain.PlanTriggerShadowUnproven {
		t.Errorf("29 days of clean shadow history gave %+v, want not enforced with %q", verdict, domain.PlanTriggerShadowUnproven)
	}
}

func TestAModeIsEnforcedOnlyWithQ1sMinimumOfEvidenceRunsInTheWindow(t *testing.T) {
	old := domain.ShadowedRun{
		At:         gatedAt.Add(-domain.ShadowWindowLength - 24*time.Hour),
		Plan:       domain.DispatchedPlan{Modes: domain.PlanModes{Selection: domain.TestModeShadow}, Recorded: domain.RecordedSets{IsRecorded: true}},
		IsReported: true,
	}
	recent := gatedAt.Add(-10 * 24 * time.Hour)
	gap := domain.ShadowedRun{At: recent, Plan: domain.DispatchedPlan{Modes: domain.PlanModes{Selection: domain.TestModeShadow}}}
	for name, bounded := range map[string]struct {
		runs []domain.ShadowedRun
		want domain.Enforcement
	}{
		"one under the minimum": {append(evidenceRuns(domain.MinimumShadowRuns-1, recent), old), domain.Enforcement{Trigger: domain.PlanTriggerShadowUnproven}},
		"at the minimum":        {append(evidenceRuns(domain.MinimumShadowRuns, recent), old), domain.Enforcement{IsEnforced: true}},
		"gaps make up the rest": {append(append(evidenceRuns(domain.MinimumShadowRuns-1, recent), gap), old), domain.Enforcement{Trigger: domain.PlanTriggerShadowUnproven}},
	} {
		if verdict := domain.EnforcementOf(domain.ShadowedSelection, bounded.runs, nil, gatedAt); verdict != bounded.want {
			t.Errorf("%s: gave %+v, want %+v", name, verdict, bounded.want)
		}
	}
}

func TestTheLatchHoldsOnlyForThisModesEscapesInsideTheWindow(t *testing.T) {
	proven := append(evidenceRuns(domain.MinimumShadowRuns, gatedAt.Add(-10*24*time.Hour)), domain.ShadowedRun{
		At:   gatedAt.Add(-domain.ShadowWindowLength - 24*time.Hour),
		Plan: domain.DispatchedPlan{Modes: domain.PlanModes{Selection: domain.TestModeShadow}},
	})
	for name, escape := range map[string]domain.Escape{
		"an escape older than the window":  {Mode: domain.ShadowedSelection, At: gatedAt.Add(-domain.ShadowWindowLength - time.Hour)},
		"an escape against the other mode": {Mode: domain.ShadowedMinimisation, At: gatedAt.Add(-time.Hour)},
	} {
		if verdict := domain.EnforcementOf(domain.ShadowedSelection, proven, []domain.Escape{escape}, gatedAt); !verdict.IsEnforced {
			t.Errorf("%s latched selection: %+v", name, verdict)
		}
	}
}
