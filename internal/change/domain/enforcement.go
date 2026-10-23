// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"slices"
	"time"
)

const ShadowWindowLength = 30 * 24 * time.Hour

const MinimumShadowRuns = 20

type Enforcement struct {
	IsEnforced bool
	Trigger    PlanTrigger
}

func EnforcementOf(mode ShadowedMode, runs []ShadowedRun, escapes []Escape, now time.Time) Enforcement {
	windowStart := now.Add(-ShadowWindowLength)
	if hasEscapeSince(mode, escapes, windowStart) {
		return Enforcement{Trigger: PlanTriggerEscapeRecorded}
	}
	if !reachesBack(mode, runs, windowStart) || evidenceSince(mode, runs, windowStart) < MinimumShadowRuns {
		return Enforcement{Trigger: PlanTriggerShadowUnproven}
	}
	return Enforcement{IsEnforced: true}
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
