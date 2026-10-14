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
	run := presentationOf(decided.Decision).checkRunFor(decided.Family, id)
	run.Text = violationsTextFor(decided.Decision.Violations())
	run.ViolationCount = len(decided.Decision.Violations())
	run.Violations = checkRunViolationsFrom(listedOf(decided.Decision.Violations()))
	return run
}

func listedOf(violations Declared[Violation]) Declared[Violation] {
	return violations[:min(len(violations), listedViolationLimit)]
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
	if decided.Family.FailurePosture() == FailurePostureOpen {
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
	run := runPresentation{
		conclusion: ConclusionNeutral,
		title:      admittedWithoutDecisionTitle,
		headline:   fmt.Sprintf(admittedWithoutDecisionLayout, markdownTextOf(string(decided.Family))),
	}.checkRunFor(decided.Family, id)
	run.Headline = fmt.Sprintf(admittedWithoutDecisionLayout, decided.Family)
	return run
}

const (
	policyErrorTitle  = "Policy failed to evaluate"
	policyErrorLayout = "Policy %s failed to evaluate. This is not a problem with your change."
)

func policyErrorCheckRun(decided FamilyDecision, id DecisionID) CheckRun {
	run := runPresentation{
		conclusion: ConclusionFailure,
		title:      policyErrorTitle,
		headline:   fmt.Sprintf(policyErrorLayout, markdownTextOf(string(decided.Family))),
	}.checkRunFor(decided.Family, id)
	run.Headline = fmt.Sprintf(policyErrorLayout, decided.Family)
	return run
}

type runPresentation struct {
	conclusion Conclusion
	title      string
	headline   string
}

const summaryLayout = "%s\n\nDecision %s"

func (presented runPresentation) checkRunFor(family PolicyFamily, id DecisionID) CheckRun {
	return CheckRun{
		Name:       checkRunNamePrefix + string(family),
		Conclusion: presented.conclusion,
		Title:      presented.title,
		Headline:   presented.headline,
		DecisionID: id,
		Summary:    fmt.Sprintf(summaryLayout, presented.headline, id),
	}
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
	listed := listedOf(violations)
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
