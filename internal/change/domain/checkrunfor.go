// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

const checkRunNamePrefix = "policy/"

type presentationRule func(decided FamilyDecision) runPresentation

var presentationRuleByOutcome = map[Outcome]presentationRule{
	OutcomeDecided:          decidedPresentation,
	OutcomeUndefined:        undecidedPresentation,
	OutcomeEngineError:      policyErrorPresentation,
	OutcomeDeadlineExceeded: policyErrorPresentation,
}

func CheckRunFor(decided FamilyDecision, id DecisionID) CheckRun {
	run := presentationFor(decided).checkRunFor(decided.Family, id)
	run.ViolationCount = len(decided.Decision.Violations())
	run.Violations = checkRunViolationsFrom(listedOf(decided.Decision.Violations()))
	return run
}

func (decided FamilyDecision) Verdict() Verdict { return presentationFor(decided).verdict }

func presentationFor(decided FamilyDecision) runPresentation {
	rule, declared := presentationRuleByOutcome[decided.Decision.Outcome()]
	if !declared {
		return policyErrorPresentation(decided)
	}
	return rule(decided)
}

func decidedPresentation(decided FamilyDecision) runPresentation {
	return presentationOf(decided.Decision)
}

func listedOf(violations Declared[Violation]) Declared[Violation] {
	return violations[:min(len(violations), ListedViolationLimit)]
}

func checkRunViolationsFrom(listed Declared[Violation]) []CheckRunViolation {
	var carried []CheckRunViolation
	for _, violation := range listed {
		carried = append(carried, CheckRunViolation{
			RuleID:     violation.RuleID,
			Message:    violation.Message,
			OwningTeam: violation.OwningTeam,
			DocsURL:    violation.DocsURL,
			Marking:    enforcementMarkingFor(violation.Severity),
		})
	}
	return carried
}

func undecidedPresentation(decided FamilyDecision) runPresentation {
	if decided.FailurePosture() == FailurePostureOpen {
		return admittedWithoutDecisionPresentation(decided)
	}
	return policyErrorPresentation(decided)
}

const (
	admittedWithoutDecisionTitle  = "Admitted without a decision"
	admittedWithoutDecisionLayout = "Policy %s did not decide, and this change was admitted " +
		"because it fails open."
)

func admittedWithoutDecisionPresentation(decided FamilyDecision) runPresentation {
	return runPresentation{
		conclusion: ConclusionNeutral,
		verdict:    VerdictAdmitted,
		title:      admittedWithoutDecisionTitle,
		headline:   fmt.Sprintf(admittedWithoutDecisionLayout, decided.Family),
	}
}

const (
	policyErrorTitle  = "Policy failed to evaluate"
	policyErrorLayout = "Policy %s failed to evaluate. This is not a problem with your change."
)

func policyErrorPresentation(decided FamilyDecision) runPresentation {
	return runPresentation{
		conclusion: ConclusionFailure,
		verdict:    VerdictRefused,
		title:      policyErrorTitle,
		headline:   fmt.Sprintf(policyErrorLayout, decided.Family),
	}
}

type runPresentation struct {
	conclusion Conclusion
	verdict    Verdict
	title      string
	headline   string
}

func (presented runPresentation) checkRunFor(family PolicyFamily, id DecisionID) CheckRun {
	return CheckRun{
		Name:       checkRunNamePrefix + string(family),
		Conclusion: presented.conclusion,
		Title:      presented.title,
		Headline:   presented.headline,
		DecisionID: id,
	}
}

var enforcementMarkingBySeverity = map[EnforcementSeverity]string{
	EnforcementSeverityDeny:   "blocking",
	EnforcementSeverityWarn:   "advisory",
	EnforcementSeverityDryRun: "recorded only",
}

func enforcementMarkingFor(severity EnforcementSeverity) string {
	marking, declared := enforcementMarkingBySeverity[severity]
	if !declared {
		return enforcementMarkingBySeverity[EnforcementSeverityDeny]
	}
	return marking
}

const ListedViolationLimit = 20
