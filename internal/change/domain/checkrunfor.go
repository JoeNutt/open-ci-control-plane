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
	presented := verdictOf(decided.Decision).presentation()
	return CheckRun{
		Name:       checkRunNameFor(decided.Family),
		Conclusion: presented.conclusion,
		Summary:    summaryWith(presented.headline, id),
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
	"admitted without a decision."

func admittedWithoutDecisionCheckRun(decided FamilyDecision, id DecisionID) CheckRun {
	return CheckRun{
		Name:       checkRunNameFor(decided.Family),
		Conclusion: ConclusionNeutral,
		Summary: summaryWith(
			fmt.Sprintf(admittedWithoutDecisionLayout, markdownTextOf(string(decided.Family))), id),
	}
}

const policyErrorLayout = "Policy %[1]s failed to evaluate. This is not a problem with your change.\n\n" +
	"Owning team: %[1]s"

func policyErrorCheckRun(decided FamilyDecision, id DecisionID) CheckRun {
	return CheckRun{
		Name:       checkRunNameFor(decided.Family),
		Conclusion: ConclusionFailure,
		Summary: summaryWith(
			fmt.Sprintf(policyErrorLayout, markdownTextOf(string(decided.Family))), id),
	}
}

func checkRunNameFor(family PolicyFamily) string {
	return checkRunNamePrefix + string(family)
}

const summaryLayout = "%s\n\nDecision %s"

func summaryWith(headline string, id DecisionID) string {
	return fmt.Sprintf(summaryLayout, headline, id)
}

var enforcementMarkingBySeverity = map[EnforcementSeverity]string{
	EnforcementSeverityDeny:   "blocking",
	EnforcementSeverityWarn:   "advisory",
	EnforcementSeverityDryRun: "recorded only, dry run",
}

func enforcementMarkingFor(severity EnforcementSeverity) string {
	marking, declared := enforcementMarkingBySeverity[severity]
	if !declared {
		return enforcementMarkingBySeverity[EnforcementSeverityDeny]
	}
	return marking
}

const violationBlockLayout = "**Rule** %s\n\n%s\n\n- Owning team: %s\n- Documentation: %s\n- Enforcement: %s\n\n"

const (
	listedViolationLimit = 20
	truncationLayout     = "and %d more — %d violations in total. This list is truncated.\n"
)

func violationsTextFor(violations Declared[Violation]) string {
	var text strings.Builder
	listed := violations[:min(len(violations), listedViolationLimit)]
	for _, violation := range listed {
		fmt.Fprintf(&text, violationBlockLayout,
			markdownTextOf(violation.RuleID),
			markdownTextOf(violation.Message),
			markdownTextOf(violation.OwningTeam),
			documentationLinkFor(violation.DocsURL),
			enforcementMarkingFor(violation.Severity))
	}
	if unlisted := len(violations) - len(listed); unlisted > 0 {
		fmt.Fprintf(&text, truncationLayout, unlisted, len(violations))
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
