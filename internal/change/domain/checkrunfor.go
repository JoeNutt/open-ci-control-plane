// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "slices"

const checkRunNamePrefix = "policy/"

func CheckRunFor(decided FamilyDecision, id DecisionID) CheckRun {
	return CheckRun{
		Name:       checkRunNamePrefix + string(decided.Family),
		Conclusion: conclusionFor(decided.Decision),
	}
}

func conclusionFor(decision Decision) Conclusion {
	if !decision.Outcome().CarriesVerdict() {
		return ConclusionFailure
	}
	if decision.Effect() == EffectBlock || isAnyDenied(decision.Violations()) {
		return ConclusionFailure
	}
	return ConclusionSuccess
}

func isAnyDenied(violations Declared[Violation]) bool {
	return slices.ContainsFunc(violations, func(violation Violation) bool {
		return violation.Severity == EnforcementSeverityDeny
	})
}
