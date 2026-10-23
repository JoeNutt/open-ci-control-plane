// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"slices"
	"time"
)

const shadowWindowLength = 30 * 24 * time.Hour

const minimumShadowRuns = 20

type enforcement struct {
	isEnforced bool
	trigger    PlanTrigger
}

func enforcementOf(mode ShadowedMode, runs []ShadowedRun, escapes []Escape, now time.Time) enforcement {
	windowStart := now.Add(-shadowWindowLength)
	if hasEscapeSince(mode, escapes, windowStart) {
		return enforcement{trigger: PlanTriggerEscapeRecorded}
	}
	if !reachesBack(mode, runs, windowStart) || evidenceSince(mode, runs, windowStart) < minimumShadowRuns {
		return enforcement{trigger: PlanTriggerShadowUnproven}
	}
	return enforcement{isEnforced: true}
}

func hasEscapeSince(mode ShadowedMode, escapes []Escape, windowStart time.Time) bool {
	return slices.ContainsFunc(escapes, func(escape Escape) bool {
		return escape.Mode == mode && escape.At.After(windowStart)
	})
}

func reachesBack(mode ShadowedMode, runs []ShadowedRun, windowStart time.Time) bool {
	return slices.ContainsFunc(runs, func(run ShadowedRun) bool {
		return run.Plan.Modes.isShadowing(mode) && !run.At.After(windowStart)
	})
}

func evidenceSince(mode ShadowedMode, runs []ShadowedRun, windowStart time.Time) int {
	evidence := 0
	for _, run := range runs {
		if run.Plan.Modes.isShadowing(mode) && run.isEvidence() && run.At.After(windowStart) {
			evidence++
		}
	}
	return evidence
}

func (plan TestPlan) IsEnforcing() bool {
	return plan.Modes.Selection == TestModeEnforce || plan.Modes.Minimisation == TestModeEnforce
}

func (plan TestPlan) GatedOn(runs []ShadowedRun, escapes []Escape, now time.Time) TestPlan {
	ownRuns, ownEscapes := runsOf(plan.Repository, runs), escapesOf(plan.Repository, escapes)
	return plan.gatedBy(
		enforcementOf(ShadowedSelection, ownRuns, ownEscapes, now),
		enforcementOf(ShadowedMinimisation, ownRuns, ownEscapes, now))
}

func (plan TestPlan) gatedBy(selection, minimisation enforcement) TestPlan {
	if plan.Modes.Selection != TestModeEnforce {
		return plan
	}
	if selection.isEnforced {
		return plan.minimisationGatedBy(minimisation)
	}
	plan.Modes.Selection = TestModeShadow
	if plan.Modes.Minimisation == TestModeEnforce {
		plan.Modes.Minimisation = TestModeShadow
	}
	shadowed := plan.shadowed()
	if shadowed.Trigger == PlanTriggerNone {
		shadowed.Trigger = selection.trigger
	}
	return shadowed
}

func (plan TestPlan) minimisationGatedBy(minimisation enforcement) TestPlan {
	if plan.Modes.Minimisation != TestModeEnforce || minimisation.isEnforced || plan.FullMatrix {
		return plan
	}
	plan.Modes.Minimisation = TestModeShadow
	plan.Recorded = plan.recordedSets()
	return plan
}
