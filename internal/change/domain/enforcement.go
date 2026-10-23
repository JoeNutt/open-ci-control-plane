// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "time"

const ShadowWindowLength = 30 * 24 * time.Hour

const MinimumShadowRuns = 20

type Enforcement struct {
	IsEnforced bool
	Trigger    PlanTrigger
}

func EnforcementOf(mode ShadowedMode, runs []ShadowedRun, escapes []Escape, now time.Time) Enforcement {
	for _, escape := range escapes {
		if escape.Mode == mode && escape.At.After(now.Add(-ShadowWindowLength)) {
			return Enforcement{Trigger: PlanTriggerEscapeRecorded}
		}
	}
	isOldEnough := false
	for _, run := range runs {
		if run.Plan.Modes.isShadowing(mode) && !run.At.After(now.Add(-ShadowWindowLength)) {
			isOldEnough = true
		}
	}
	evidence := 0
	for _, run := range runs {
		if run.Plan.Modes.isShadowing(mode) && run.isEvidence() && run.At.After(now.Add(-ShadowWindowLength)) {
			evidence++
		}
	}
	if !isOldEnough || evidence < MinimumShadowRuns {
		return Enforcement{Trigger: PlanTriggerShadowUnproven}
	}
	return Enforcement{IsEnforced: true}
}
