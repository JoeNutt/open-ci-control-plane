// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
	"strings"
)

const checkRunNamePrefix = "policy/"

var conclusionByPosture = map[FailurePosture]Conclusion{
	FailurePostureOpen:              ConclusionNeutral,
	FailurePostureClosed:            ConclusionFailure,
	FailurePostureDeclaredPerPolicy: ConclusionFailure,
}

var conclusionByEffect = map[Effect]Conclusion{
	EffectBlock:  ConclusionFailure,
	EffectAllow:  ConclusionSuccess,
	EffectReview: ConclusionSuccess,
}

var conclusionBySeverity = map[EnforcementSeverity]Conclusion{
	EnforcementSeverityDeny:   ConclusionFailure,
	EnforcementSeverityWarn:   ConclusionNeutral,
	EnforcementSeverityDryRun: ConclusionNeutral,
}

var conclusionsLeastRestrictiveFirst = []Conclusion{
	ConclusionSuccess,
	ConclusionNeutral,
	ConclusionFailure,
}

func CheckRunFor(decided FamilyDecision, id DecisionID) CheckRun {
	if decided.Decision.Outcome() == OutcomeEngineError {
		return policyErrorCheckRun(decided.Family, id)
	}
	return CheckRun{
		Name:       checkRunNameFor(decided.Family),
		Conclusion: conclusionFor(decided),
		Summary:    decisionLineFor(id),
		Text:       violationsTextFor(decided.Decision.Violations()),
	}
}

func checkRunNameFor(family PolicyFamily) string {
	return checkRunNamePrefix + string(family)
}

const policyErrorLayout = "Policy %[1]s failed to evaluate. This is not a problem with your change.\n\n" +
	"Owning team: %[1]s\n\n%[2]s"

func policyErrorCheckRun(family PolicyFamily, id DecisionID) CheckRun {
	return CheckRun{
		Name:       checkRunNameFor(family),
		Conclusion: ConclusionFailure,
		Summary:    fmt.Sprintf(policyErrorLayout, markdownTextOf(string(family)), decisionLineFor(id)),
	}
}

const decisionLineLayout = "Decision %s"

func decisionLineFor(id DecisionID) string {
	return fmt.Sprintf(decisionLineLayout, id)
}

const violationBlockLayout = "**Rule** %s\n\n%s\n\n- Owning team: %s\n- Documentation: %s\n\n"

const withheldDocumentationLink = "withheld, not an https link"

func documentationLinkFor(stated string) string {
	link, isHTTPS := markdownHTTPSAutolinkOf(stated)
	if !isHTTPS {
		return withheldDocumentationLink
	}
	return link
}

func violationsTextFor(violations Declared[Violation]) string {
	var text strings.Builder
	for _, violation := range violations {
		fmt.Fprintf(&text, violationBlockLayout,
			markdownTextOf(violation.RuleID),
			markdownTextOf(violation.Message),
			markdownTextOf(violation.OwningTeam),
			documentationLinkFor(violation.DocsURL))
	}
	return text.String()
}

func conclusionFor(decided FamilyDecision) Conclusion {
	switch decided.Decision.Outcome() {
	case OutcomeDecided:
		return verdictConclusionFor(decided.Decision)
	case OutcomeUndefined:
		return conclusionIn(conclusionByPosture, decided.Family.FailurePosture())
	case OutcomeEngineError, OutcomeDeadlineExceeded:
		return ConclusionFailure
	default:
		return ConclusionFailure
	}
}

func verdictConclusionFor(decision Decision) Conclusion {
	conclusion := conclusionIn(conclusionByEffect, decision.Effect())
	for _, violation := range decision.Violations() {
		conclusion = moreRestrictive(conclusion, conclusionIn(conclusionBySeverity, violation.Severity))
	}
	return conclusion
}

func conclusionIn[Key comparable](conclusions map[Key]Conclusion, key Key) Conclusion {
	conclusion, declared := conclusions[key]
	if !declared {
		return ConclusionFailure
	}
	return conclusion
}

func moreRestrictive(held, offered Conclusion) Conclusion {
	if slices.Index(conclusionsLeastRestrictiveFirst, offered) >
		slices.Index(conclusionsLeastRestrictiveFirst, held) {
		return offered
	}
	return held
}
