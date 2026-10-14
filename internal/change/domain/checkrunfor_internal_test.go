// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "testing"

func TestAnUndeclaredEffectOrSeverityFailsClosed(t *testing.T) {
	undeclared := map[string]Decision{
		"an undeclared effect": {outcome: OutcomeDecided, effect: "admit"},
		"an undeclared severity": {
			outcome:    OutcomeDecided,
			effect:     EffectAllow,
			violations: Declared[Violation]{{RuleID: "AUTHZ-1", Severity: "advise"}},
		},
	}

	for form, decision := range undeclared {
		t.Run(form, func(t *testing.T) {
			run := CheckRunFor(FamilyDecision{Family: PolicyFamilySecurity, Decision: decision}, DecisionID{})

			if run.Conclusion != ConclusionFailure {
				t.Errorf("conclusion is %q, want %q", run.Conclusion, ConclusionFailure)
			}
		})
	}
}
