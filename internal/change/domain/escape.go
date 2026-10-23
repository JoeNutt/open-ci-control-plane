// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "slices"

type ShadowedMode string

const (
	ShadowedSelection    ShadowedMode = "selection"
	ShadowedMinimisation ShadowedMode = "minimisation"
)

type Escape struct {
	Mode         ShadowedMode
	PullRequest  PullRequestLocator
	Test         TestResult
	ChangedFiles []RepositoryPath
}

func (dispatched DispatchedPlan) EscapesIn(results []TestResult) []Escape {
	var escapes []Escape
	for _, shadowed := range dispatched.shadowedSets() {
		for _, escaped := range dispatched.escapesOutside(shadowed.recorded, results) {
			escapes = append(escapes, Escape{
				Mode: shadowed.mode, PullRequest: dispatched.Locator, Test: escaped, ChangedFiles: dispatched.ChangedFiles,
			})
		}
	}
	return escapes
}

type shadowedSet struct {
	mode     ShadowedMode
	recorded []CoverageSuite
}

func (dispatched DispatchedPlan) shadowedSets() []shadowedSet {
	if !dispatched.Recorded.IsRecorded {
		return nil
	}
	sets := []shadowedSet{{mode: ShadowedSelection, recorded: dispatched.Recorded.Selected}}
	if dispatched.Modes.Minimisation == TestModeShadow {
		sets = append(sets, shadowedSet{mode: ShadowedMinimisation, recorded: dispatched.Recorded.Minimised})
	}
	return sets
}

func (dispatched DispatchedPlan) escapesOutside(recorded []CoverageSuite, results []TestResult) []TestResult {
	var escaped []TestResult
	for _, result := range results {
		if result.failedTwice() && !dispatched.wouldHaveRun(recorded, result.Suite) {
			escaped = append(escaped, result)
		}
	}
	return escaped
}

func (dispatched DispatchedPlan) wouldHaveRun(recorded []CoverageSuite, suite CoverageSuite) bool {
	return slices.Contains(recorded, suite) ||
		slices.ContainsFunc(dispatched.AlwaysRun, func(path GoImportPath) bool { return path.String() == suite.String() })
}

func (result TestResult) failedTwice() bool {
	return result.Outcome == TestOutcomeFail && result.Rerun == TestOutcomeFail
}
