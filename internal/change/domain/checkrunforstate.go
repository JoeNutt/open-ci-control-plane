// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type violationGrade int

const (
	gradeNone violationGrade = iota
	gradeRecorded
	gradeAdvisory
	gradeBlocking
)

var gradeBySeverity = map[EnforcementSeverity]violationGrade{
	EnforcementSeverityDeny:   gradeBlocking,
	EnforcementSeverityWarn:   gradeAdvisory,
	EnforcementSeverityDryRun: gradeRecorded,
}

type decidedState struct {
	effect Effect
	grade  violationGrade
}

var (
	passed = runPresentation{
		conclusion: ConclusionSuccess, verdict: VerdictAdmitted, title: "Passed", headline: "Passed. No rule fired.",
	}
	recordedOnly = runPresentation{
		conclusion: ConclusionNeutral, verdict: VerdictAdmitted, title: "Recorded only",
		headline: "Recorded only. This policy is in dry run and did not affect your pull request.",
	}
	advisory = runPresentation{
		conclusion: ConclusionNeutral, verdict: VerdictAdmitted, title: "Advisory",
		headline: "Advisory only. The merge is not prevented.",
	}
	blockedByRules = runPresentation{
		conclusion: ConclusionFailure, verdict: VerdictRefused, title: "Blocked", headline: "Blocked by the rules below.",
	}
	blockedNamingNoRule = runPresentation{
		conclusion: ConclusionFailure, verdict: VerdictRefused, title: "Blocked", headline: "Blocked. The policy named no rule.",
	}
	blockedDespiteViolations = runPresentation{
		conclusion: ConclusionFailure, verdict: VerdictRefused, title: "Blocked",
		headline: "Blocked by the policy. The violations below did not cause the block.",
	}
	reviewRequired = runPresentation{
		conclusion: ConclusionSuccess, verdict: VerdictReviewRequired, title: "Review required", headline: "Review required. No rule fired.",
	}
	reviewRequiredWithRecorded = runPresentation{
		conclusion: ConclusionNeutral, verdict: VerdictReviewRequired, title: "Review required",
		headline: "Review required. The violations below are recorded only, from a policy in dry run.",
	}
	reviewRequiredWithAdvisory = runPresentation{
		conclusion: ConclusionNeutral, verdict: VerdictReviewRequired, title: "Review required",
		headline: "Review required. The violations below are advisory or recorded only.",
	}
)

var presentationByState = map[decidedState]runPresentation{
	{EffectAllow, gradeNone}:      passed,
	{EffectAllow, gradeRecorded}:  recordedOnly,
	{EffectAllow, gradeAdvisory}:  advisory,
	{EffectAllow, gradeBlocking}:  blockedByRules,
	{EffectReview, gradeNone}:     reviewRequired,
	{EffectReview, gradeRecorded}: reviewRequiredWithRecorded,
	{EffectReview, gradeAdvisory}: reviewRequiredWithAdvisory,
	{EffectReview, gradeBlocking}: blockedByRules,
	{EffectBlock, gradeNone}:      blockedNamingNoRule,
	{EffectBlock, gradeRecorded}:  blockedDespiteViolations,
	{EffectBlock, gradeAdvisory}:  blockedDespiteViolations,
	{EffectBlock, gradeBlocking}:  blockedByRules,
}

func presentationOf(decision Decision) runPresentation {
	grade := gradeOf(decision.Violations())
	presented, declared := presentationByState[decidedState{decision.Effect(), grade}]
	if !declared {
		return presentationByState[decidedState{EffectBlock, grade}]
	}
	return presented
}

func gradeOf(violations Declared[Violation]) violationGrade {
	reached := gradeNone
	for _, violation := range violations {
		reached = max(reached, gradeIn(violation.Severity))
	}
	return reached
}

func gradeIn(severity EnforcementSeverity) violationGrade {
	grade, declared := gradeBySeverity[severity]
	if !declared {
		return gradeBlocking
	}
	return grade
}
