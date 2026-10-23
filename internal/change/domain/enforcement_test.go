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
