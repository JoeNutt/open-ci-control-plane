// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"strings"
)

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
	summary := decisionLineFor(id)
	if verdictConclusionFor(decided.Decision) == ConclusionNeutral {
		summary = "Advisory only. The merge is not prevented.\n\n" + summary
	}
	return CheckRun{
		Name:       checkRunNameFor(decided.Family),
		Conclusion: verdictConclusionFor(decided.Decision),
		Summary:    summary,
		Text:       violationsTextFor(decided.Decision.Violations()),
	}
}

var undecidedRenderingByPosture = map[FailurePosture]checkRunRendering{
	FailurePostureOpen:              admittedWithoutDecisionCheckRun,
	FailurePostureClosed:            policyErrorCheckRun,
	FailurePostureDeclaredPerPolicy: policyErrorCheckRun,
}

func undecidedCheckRun(decided FamilyDecision, id DecisionID) CheckRun {
	rendering, declared := undecidedRenderingByPosture[decided.Family.FailurePosture()]
	if !declared {
		return policyErrorCheckRun(decided, id)
	}
	return rendering(decided, id)
}

const admittedWithoutDecisionLayout = "Policy %s could not be evaluated and this change was " +
	"admitted without a decision.\n\n%s"

func admittedWithoutDecisionCheckRun(decided FamilyDecision, id DecisionID) CheckRun {
	return CheckRun{
		Name:       checkRunNameFor(decided.Family),
		Conclusion: ConclusionNeutral,
		Summary: fmt.Sprintf(admittedWithoutDecisionLayout,
			markdownTextOf(string(decided.Family)), decisionLineFor(id)),
	}
}

const policyErrorLayout = "Policy %[1]s failed to evaluate. This is not a problem with your change.\n\n" +
	"Owning team: %[1]s\n\n%[2]s"

func policyErrorCheckRun(decided FamilyDecision, id DecisionID) CheckRun {
	return CheckRun{
		Name:       checkRunNameFor(decided.Family),
		Conclusion: ConclusionFailure,
		Summary: fmt.Sprintf(policyErrorLayout,
			markdownTextOf(string(decided.Family)), decisionLineFor(id)),
	}
}

func checkRunNameFor(family PolicyFamily) string {
	return checkRunNamePrefix + string(family)
}

const decisionLineLayout = "Decision %s"

func decisionLineFor(id DecisionID) string {
	return fmt.Sprintf(decisionLineLayout, id)
}

const violationBlockLayout = "**Rule** %s\n\n%s\n\n- Owning team: %s\n- Documentation: %s\n- Enforcement: %s\n\n"

func violationsTextFor(violations Declared[Violation]) string {
	var text strings.Builder
	for _, violation := range violations {
		fmt.Fprintf(&text, violationBlockLayout,
			markdownTextOf(violation.RuleID),
			markdownTextOf(violation.Message),
			markdownTextOf(violation.OwningTeam),
			documentationLinkFor(violation.DocsURL),
			map[EnforcementSeverity]string{EnforcementSeverityWarn: "advisory"}[violation.Severity])
	}
	return text.String()
}

const withheldDocumentationLink = "withheld, not an https link"

func documentationLinkFor(stated string) string {
	link, isHTTPS := markdownHTTPSAutolinkOf(stated)
	if !isHTTPS {
		return withheldDocumentationLink
	}
	return link
}
