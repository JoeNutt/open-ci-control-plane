// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestARunWithNoPlanOrNoResultsIsAGapNeverZeroEscapes(t *testing.T) {
	inShadow := domain.PlanModes{Selection: domain.TestModeShadow, Minimisation: domain.TestModeOff}
	recorded := domain.RecordedSets{IsRecorded: true}
	runs := []domain.ShadowedRun{
		{Plan: domain.DispatchedPlan{Modes: inShadow, Recorded: recorded}, IsReported: true},
		{Plan: domain.DispatchedPlan{Modes: inShadow, Recorded: recorded}, IsReported: false},
		{Plan: domain.DispatchedPlan{Modes: inShadow}, IsReported: true},
	}

	window := domain.ShadowWindowOf(domain.ShadowedSelection, runs, nil)

	if window.Runs != 3 || window.Gaps != 2 || window.Escapes != 0 {
		t.Errorf("the window counts %+v, want 3 runs, 2 gaps and 0 escapes", window)
	}
}

func TestOnlyEscapesAgainstTheCountedModeAreCounted(t *testing.T) {
	escapes := []domain.Escape{{Mode: domain.ShadowedSelection}, {Mode: domain.ShadowedMinimisation}, {Mode: domain.ShadowedSelection}}

	if window := domain.ShadowWindowOf(domain.ShadowedSelection, nil, escapes); window.Escapes != 2 {
		t.Errorf("selection counts %d escapes, want 2", window.Escapes)
	}
	if window := domain.ShadowWindowOf(domain.ShadowedMinimisation, nil, escapes); window.Escapes != 1 {
		t.Errorf("minimisation counts %d escapes, want 1", window.Escapes)
	}
}

func TestAModesWindowCountsOnlyTheRunsThatShadowedIt(t *testing.T) {
	recorded := domain.RecordedSets{IsRecorded: true}
	runs := []domain.ShadowedRun{
		{Plan: domain.DispatchedPlan{Modes: domain.PlanModes{Selection: domain.TestModeShadow, Minimisation: domain.TestModeShadow}, Recorded: recorded}, IsReported: true},
		{Plan: domain.DispatchedPlan{Modes: domain.PlanModes{Selection: domain.TestModeShadow, Minimisation: domain.TestModeOff}, Recorded: recorded}, IsReported: true},
		{Plan: domain.DispatchedPlan{Modes: domain.PlanModes{Selection: domain.TestModeEnforce, Minimisation: domain.TestModeOff}}, IsReported: true},
	}

	if window := domain.ShadowWindowOf(domain.ShadowedSelection, runs, nil); window.Runs != 2 || window.Gaps != 0 {
		t.Errorf("selection counts %+v, want 2 runs and no gap", window)
	}
	if window := domain.ShadowWindowOf(domain.ShadowedMinimisation, runs, nil); window.Runs != 1 || window.Gaps != 0 {
		t.Errorf("minimisation counts %+v, want 1 run and no gap", window)
	}
}
