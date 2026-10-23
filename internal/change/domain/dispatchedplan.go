// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
)

const (
	testsPassedTitle      = "Every planned test passed"
	testsFailedTitle      = "A planned test failed or did not report"
	testsReportedHeadline = "%d tests reported."
)

type DispatchedPlan struct {
	Locator      PullRequestLocator
	Head         CommitSHA
	DecisionID   DecisionID
	Suites       []CoverageSuite
	AlwaysRun    []GoImportPath
	Recorded     RecordedSets
	ChangedFiles []RepositoryPath
}

func DispatchedPlanOf(locator PullRequestLocator, plan TestPlan) DispatchedPlan {
	return DispatchedPlan{
		Locator: locator, Head: plan.Head, DecisionID: plan.DecisionID, Suites: plan.Suites,
		AlwaysRun: plan.AlwaysRun, Recorded: plan.Recorded, ChangedFiles: plan.ChangedFiles,
	}
}

func (dispatched DispatchedPlan) IsReportedBy(report RunReport) bool {
	return dispatched.DecisionID == report.DecisionID()
}

func (dispatched DispatchedPlan) TestsCheckRunOf(results []TestResult) CheckRun {
	conclusion := dispatched.ConclusionOf(results)
	title := testsFailedTitle
	if conclusion == ConclusionSuccess {
		title = testsPassedTitle
	}
	return CheckRun{
		Name: TestsCheckRunName, Conclusion: conclusion, Title: title,
		Headline: fmt.Sprintf(testsReportedHeadline, len(results)), DecisionID: dispatched.DecisionID,
	}
}

func (dispatched DispatchedPlan) ConclusionOf(results []TestResult) Conclusion {
	if len(results) == 0 {
		return ConclusionFailure
	}
	reported := map[CoverageSuite]bool{}
	for _, result := range results {
		if result.Outcome == TestOutcomeFail {
			return ConclusionFailure
		}
		reported[result.Suite] = true
	}
	for _, suite := range dispatched.Suites {
		if !reported[suite] {
			return ConclusionFailure
		}
	}
	return ConclusionSuccess
}

func (dispatched DispatchedPlan) SelectionEscapesIn(results []TestResult) []TestResult {
	return dispatched.escapesOutside(dispatched.Recorded.Selected, results)
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
