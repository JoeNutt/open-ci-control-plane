// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "testing"

func TestAnUndeclaredEffectOrSeverityFailsClosed(t *testing.T) {
	undeclared := map[string]Decision{
		"an undeclared effect":   {outcome: OutcomeDecided, effect: "admit"},
		"an undeclared severity": undeclaredSeverityDecision(),
	}

	for form, decision := range undeclared {
		t.Run(form, func(t *testing.T) {
			run := securityCheckRunFor(decision)

			if run.Conclusion != ConclusionFailure {
				t.Errorf("conclusion is %q, want %q", run.Conclusion, ConclusionFailure)
			}
		})
	}
}

func TestAnUndeclaredSeverityIsMarkedBlocking(t *testing.T) {
	run := securityCheckRunFor(undeclaredSeverityDecision())

	if marked := "blocking"; run.Violations[0].Marking != marked {
		t.Errorf("marking is %q, want %q", run.Violations[0].Marking, marked)
	}
}

func undeclaredSeverityDecision() Decision {
	return Decision{
		outcome:    OutcomeDecided,
		effect:     EffectAllow,
		violations: Declared[Violation]{{RuleID: "AUTHZ-1", Severity: "advise"}},
	}
}

func securityCheckRunFor(decision Decision) CheckRun {
	return CheckRunFor(FamilyDecision{Family: PolicyFamilySecurity, Decision: decision}, DecisionID{})
}
