// SPDX-License-Identifier: GPL-3.0-or-later

package domain

const checkRunNamePrefix = "policy/"

func CheckRunFor(decided FamilyDecision, id DecisionID) CheckRun {
	conclusion := ConclusionSuccess
	if !decided.Decision.Outcome().CarriesVerdict() || decided.Decision.Effect() == EffectBlock {
		conclusion = ConclusionFailure
	}
	for _, violation := range decided.Decision.Violations() {
		if violation.Severity == EnforcementSeverityDeny {
			conclusion = ConclusionFailure
		}
	}
	return CheckRun{Name: checkRunNamePrefix + string(decided.Family), Conclusion: conclusion}
}
