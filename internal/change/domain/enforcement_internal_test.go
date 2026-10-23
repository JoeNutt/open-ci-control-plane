// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"testing"
	"time"
)

var heldAt = time.Date(2026, 11, 30, 12, 0, 0, 0, time.UTC)

var windowStart = heldAt.Add(-shadowWindowLength)

func runAt(at time.Time, isRecorded bool) ShadowedRun {
	inShadow := PlanModes{Selection: TestModeShadow, Minimisation: TestModeShadow}
	return ShadowedRun{At: at, Plan: DispatchedPlan{Modes: inShadow, Recorded: RecordedSets{IsRecorded: isRecorded}}, IsReported: true}
}

func evidenceRunsFrom(oldest time.Time, count int) []ShadowedRun {
	runs := make([]ShadowedRun, 0, count)
	for index := range count {
		runs = append(runs, runAt(oldest.Add(time.Duration(index)*time.Minute), true))
	}
	return runs
}

func TestASelectionHoldsUntilItsShadowHistoryReachesBackTheWholeWindow(t *testing.T) {
	for name, bounded := range map[string]struct {
		oldest time.Time
		want   PlanTrigger
	}{
		"an hour short of the window": {windowStart.Add(time.Hour), PlanTriggerShadowUnproven},
		"exactly the window":          {windowStart, PlanTriggerNone},
	} {
		runs := append(evidenceRunsFrom(heldAt.Add(-time.Hour), minimumShadowRuns), runAt(bounded.oldest, false))
		if hold := shadowHoldOn(ShadowedSelection, runs, nil, heldAt); hold != bounded.want {
			t.Errorf("%s: held by %q, want %q", name, hold, bounded.want)
		}
	}
}

func TestASelectionHoldsUntilTheWindowHoldsQ1sMinimumOfEvidenceRuns(t *testing.T) {
	reachingBack := runAt(windowStart.Add(-time.Hour), true)
	gap := runAt(heldAt.Add(-time.Hour), false)
	for name, bounded := range map[string]struct {
		runs []ShadowedRun
		want PlanTrigger
	}{
		"one under the minimum": {append(evidenceRunsFrom(heldAt.Add(-time.Hour), minimumShadowRuns-1), reachingBack), PlanTriggerShadowUnproven},
		"at the minimum":        {append(evidenceRunsFrom(heldAt.Add(-time.Hour), minimumShadowRuns), reachingBack), PlanTriggerNone},
		"a gap makes up the rest": {
			append(evidenceRunsFrom(heldAt.Add(-time.Hour), minimumShadowRuns-1), gap, reachingBack), PlanTriggerShadowUnproven,
		},
	} {
		if hold := shadowHoldOn(ShadowedSelection, bounded.runs, nil, heldAt); hold != bounded.want {
			t.Errorf("%s: held by %q, want %q", name, hold, bounded.want)
		}
	}
}

func TestAnEscapeHoldsItsModeOnlyWhileItIsInsideTheWindow(t *testing.T) {
	proven := append(evidenceRunsFrom(heldAt.Add(-time.Hour), minimumShadowRuns), runAt(windowStart, false))
	for name, bounded := range map[string]struct {
		escape Escape
		want   PlanTrigger
	}{
		"an hour inside the window": {Escape{Mode: ShadowedSelection, At: windowStart.Add(time.Hour)}, PlanTriggerEscapeRecorded},
		"at the window's start":     {Escape{Mode: ShadowedSelection, At: windowStart}, PlanTriggerNone},
		"against the other mode":    {Escape{Mode: ShadowedMinimisation, At: heldAt.Add(-time.Hour)}, PlanTriggerNone},
	} {
		if hold := shadowHoldOn(ShadowedSelection, proven, []Escape{bounded.escape}, heldAt); hold != bounded.want {
			t.Errorf("%s: held by %q, want %q", name, hold, bounded.want)
		}
	}
}
