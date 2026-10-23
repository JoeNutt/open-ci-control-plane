// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"slices"
	"time"
)

const shadowWindowLength = 30 * 24 * time.Hour

const minimumShadowRuns = 20

func (plan TestPlan) IsEnforcing() bool {
	return plan.Modes.Selection == TestModeEnforce || plan.Modes.Minimisation == TestModeEnforce
}

func (plan TestPlan) GatedOn(runs []ShadowedRun, escapes []Escape, now time.Time) TestPlan {
	ownRuns, ownEscapes := runsOf(plan.Repository, runs), escapesOf(plan.Repository, escapes)
	return plan.gatedBy(
		shadowHoldOn(ShadowedSelection, ownRuns, ownEscapes, now),
		shadowHoldOn(ShadowedMinimisation, ownRuns, ownEscapes, now))
}

func (plan TestPlan) gatedBy(selectionHold, minimisationHold PlanTrigger) TestPlan {
	if plan.Modes.Selection != TestModeEnforce {
		return plan
	}
	if selectionHold == PlanTriggerNone {
		return plan.minimisationGatedBy(minimisationHold)
	}
	plan.Modes.Selection = TestModeShadow
	if plan.Modes.Minimisation == TestModeEnforce {
		plan.Modes.Minimisation = TestModeShadow
	}
	shadowed := plan.shadowed()
	if shadowed.Trigger == PlanTriggerNone {
		shadowed.Trigger = selectionHold
	}
	return shadowed
}

func (plan TestPlan) minimisationGatedBy(minimisationHold PlanTrigger) TestPlan {
	if plan.Modes.Minimisation != TestModeEnforce || minimisationHold == PlanTriggerNone || plan.FullMatrix {
		return plan
	}
	plan.Modes.Minimisation = TestModeShadow
	plan.Recorded = plan.recordedSets()
	return plan
}

func shadowHoldOn(mode ShadowedMode, runs []ShadowedRun, escapes []Escape, now time.Time) PlanTrigger {
	windowStart := now.Add(-shadowWindowLength)
	if hasEscapeSince(mode, escapes, windowStart) {
		return PlanTriggerEscapeRecorded
	}
	if !historyReachesBack(mode, runs, windowStart) || evidenceCountSince(mode, runs, windowStart) < minimumShadowRuns {
		return PlanTriggerShadowUnproven
	}
	return PlanTriggerNone
}

func hasEscapeSince(mode ShadowedMode, escapes []Escape, windowStart time.Time) bool {
	return slices.ContainsFunc(escapes, func(escape Escape) bool {
		return escape.Mode == mode && escape.At.After(windowStart)
	})
}

func historyReachesBack(mode ShadowedMode, runs []ShadowedRun, windowStart time.Time) bool {
	return slices.ContainsFunc(runs, func(run ShadowedRun) bool {
		return run.Plan.Modes.isShadowing(mode) && !run.At.After(windowStart)
	})
}

func evidenceCountSince(mode ShadowedMode, runs []ShadowedRun, windowStart time.Time) int {
	evidence := 0
	for _, run := range runs {
		if run.Plan.Modes.isShadowing(mode) && run.isEvidence() && run.At.After(windowStart) {
			evidence++
		}
	}
	return evidence
}
