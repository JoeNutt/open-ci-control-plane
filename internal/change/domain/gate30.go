// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "time"

type Enforcement struct {
	IsEnforced bool
	Trigger    PlanTrigger
}

func EnforcementOf(mode ShadowedMode, runs []ShadowedRun, escapes []Escape, now time.Time) Enforcement {
	for _, escape := range escapes {
		if escape.Mode == mode && escape.At.After(now.Add(-30*24*time.Hour)) {
			return Enforcement{Trigger: PlanTriggerEscapeRecorded}
		}
	}
	return Enforcement{IsEnforced: true}
}
