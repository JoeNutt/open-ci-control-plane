// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

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
	Tests        []PackageTests
	AlwaysRun    []GoImportPath
	Recorded     RecordedSets
	ChangedFiles []RepositoryPath
	Modes        PlanModes
}

func DispatchedPlanOf(locator PullRequestLocator, plan TestPlan) DispatchedPlan {
	return DispatchedPlan{
		Locator: locator, Head: plan.Head, DecisionID: plan.DecisionID, Suites: plan.Suites, Tests: plan.Tests,
		AlwaysRun: plan.AlwaysRun, Recorded: plan.Recorded, ChangedFiles: plan.ChangedFiles, Modes: plan.Modes,
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
	reportedPackages := map[GoImportPath]bool{}
	for _, result := range results {
		if result.Outcome == TestOutcomeFail {
			return ConclusionFailure
		}
		reported[result.Suite] = true
		reportedPackages[result.Suite.importPath] = true
	}
	for _, entry := range dispatched.Tests {
		if !reportedPackages[entry.pkg] {
			return ConclusionFailure
		}
	}
	for _, suite := range dispatched.Suites {
		if !reported[suite] {
			return ConclusionFailure
		}
	}
	return ConclusionSuccess
}
