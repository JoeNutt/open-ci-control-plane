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

type verdictPresentation struct {
	conclusion Conclusion
	headline   string
}

const (
	recordedOnlyHeadline = "Recorded only. This policy is in dry run and did not affect your pull request."
	advisoryHeadline     = "Advisory only. The merge is not prevented."
)

var presentationByVerdict = [...]verdictPresentation{
	verdictPassed:   {conclusion: ConclusionSuccess},
	verdictRecorded: {conclusion: ConclusionNeutral, headline: recordedOnlyHeadline},
	verdictWarned:   {conclusion: ConclusionNeutral, headline: advisoryHeadline},
	verdictBlocked:  {conclusion: ConclusionFailure},
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

func (reached verdict) presentation() verdictPresentation {
	return presentationByVerdict[reached]
}
