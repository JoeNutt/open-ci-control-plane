// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

const checkRunNamePrefix = "policy/"

type checkRunRendering func(decided FamilyDecision, id DecisionID) CheckRun

var checkRunRenderingByOutcome = map[Outcome]checkRunRendering{
	OutcomeDecided:          verdictCheckRun,
	OutcomeUndefined:        undecidedCheckRun,
	OutcomeEngineError:      policyErrorCheckRun,
	OutcomeDeadlineExceeded: policyErrorCheckRun,
}

func CheckRunFor(decided FamilyDecision, id DecisionID) CheckRun {
	rendering, declared := checkRunRenderingByOutcome[decided.Decision.Outcome()]
	if !declared {
		return policyErrorCheckRun(decided, id)
	}
	return rendering(decided, id)
}

func verdictCheckRun(decided FamilyDecision, id DecisionID) CheckRun {
	run := presentationOf(decided.Decision).checkRunFor(decided.Family, id)
	run.ViolationCount = len(decided.Decision.Violations())
	run.Violations = checkRunViolationsFrom(listedOf(decided.Decision.Violations()))
	return run
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

func undecidedCheckRun(decided FamilyDecision, id DecisionID) CheckRun {
	posture := decided.Decision.FailurePosture()
	if posture == "" {
		posture = decided.Family.FailurePosture()
	}
	if posture == FailurePostureOpen {
		return admittedWithoutDecisionCheckRun(decided, id)
	}
	return policyErrorCheckRun(decided, id)
}

const (
	admittedWithoutDecisionTitle  = "Admitted without a decision"
	admittedWithoutDecisionLayout = "Policy %s could not be evaluated and this change was " +
		"admitted without a decision."
)

func admittedWithoutDecisionCheckRun(decided FamilyDecision, id DecisionID) CheckRun {
	return runPresentation{
		conclusion: ConclusionNeutral,
		title:      admittedWithoutDecisionTitle,
		headline:   fmt.Sprintf(admittedWithoutDecisionLayout, decided.Family),
	}.checkRunFor(decided.Family, id)
}

const (
	policyErrorTitle  = "Policy failed to evaluate"
	policyErrorLayout = "Policy %s failed to evaluate. This is not a problem with your change."
)

func policyErrorCheckRun(decided FamilyDecision, id DecisionID) CheckRun {
	return runPresentation{
		conclusion: ConclusionFailure,
		title:      policyErrorTitle,
		headline:   fmt.Sprintf(policyErrorLayout, decided.Family),
	}.checkRunFor(decided.Family, id)
}

type runPresentation struct {
	conclusion Conclusion
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
