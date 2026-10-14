// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type verdict int

const (
	verdictPassed verdict = iota
	verdictRecorded
	verdictWarned
	verdictBlocked
)

var verdictByEffect = map[Effect]verdict{
	EffectBlock:  verdictBlocked,
	EffectAllow:  verdictPassed,
	EffectReview: verdictPassed,
}

var verdictBySeverity = map[EnforcementSeverity]verdict{
	EnforcementSeverityDeny:   verdictBlocked,
	EnforcementSeverityWarn:   verdictWarned,
	EnforcementSeverityDryRun: verdictRecorded,
}

var presentationByVerdict = [...]runPresentation{
	verdictPassed: {
		conclusion: ConclusionSuccess,
		title:      "Passed",
		headline:   "Passed. No rule fired.",
	},
	verdictRecorded: {
		conclusion: ConclusionNeutral,
		title:      "Recorded only",
		headline:   "Recorded only. This policy is in dry run and did not affect your pull request.",
	},
	verdictWarned: {
		conclusion: ConclusionNeutral,
		title:      "Advisory",
		headline:   "Advisory only. The merge is not prevented.",
	},
	verdictBlocked: {
		conclusion: ConclusionFailure,
		title:      "Blocked",
		headline:   "Blocked by the rules below.",
	},
}

func verdictOf(decision Decision) verdict {
	reached := verdictIn(verdictByEffect, decision.Effect())
	for _, violation := range decision.Violations() {
		reached = max(reached, verdictIn(verdictBySeverity, violation.Severity))
	}
	return reached
}

func verdictIn[Key comparable](verdicts map[Key]verdict, key Key) verdict {
	stated, declared := verdicts[key]
	if !declared {
		return verdictBlocked
	}
	return stated
}

func (reached verdict) presentation() runPresentation {
	return presentationByVerdict[reached]
}
