// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"context"
	"errors"
	"fmt"
)

var ErrNoPlanForHead = errors.New("no plan was dispatched for this head")

var ErrRunAlreadyReported = errors.New("this head's run was already reported")

type DispatchedPlan struct {
	Locator    PullRequestLocator
	Head       CommitSHA
	DecisionID DecisionID
	Suites     []CoverageSuite
}

func DispatchedPlanOf(locator PullRequestLocator, plan TestPlan) DispatchedPlan {
	return DispatchedPlan{Locator: locator, Head: plan.Head, DecisionID: plan.DecisionID, Suites: plan.Suites}
}

func (dispatched DispatchedPlan) IsReportedBy(report RunReport) bool {
	return dispatched.DecisionID == report.DecisionID()
}

func (dispatched DispatchedPlan) TestsCheckRunOf(results []TestResult) CheckRun {
	conclusion := dispatched.ConclusionOf(results)
	title := "A planned test failed or did not report"
	if conclusion == ConclusionSuccess {
		title = "Every planned test passed"
	}
	return CheckRun{
		Name: TestsCheckRunName, Conclusion: conclusion, Title: title,
		Headline: fmt.Sprintf("%d tests reported.", len(results)), DecisionID: dispatched.DecisionID,
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

type PlanLedger interface {
	DispatchedPlanFor(ctx context.Context, repository GovernedRepository, head CommitSHA) (DispatchedPlan, error)
	RecordReport(ctx context.Context, repository GovernedRepository, head CommitSHA) error
	RecordDispatch(ctx context.Context, dispatched DispatchedPlan) error
}
