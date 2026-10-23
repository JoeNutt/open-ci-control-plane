// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

const TestsCheckRunName = "open-ci/tests"

type TestOutcome string

const (
	TestOutcomePass TestOutcome = "pass"
	TestOutcomeFail TestOutcome = "fail"
)

type TestResult struct {
	Suite   CoverageSuite
	Name    string
	Outcome TestOutcome
}

type RunReport struct {
	Repository GovernedRepository
	Head       CommitSHA
	Results    []TestResult
}

const MaxReportedTests = 10000

var ErrTooManyReportedTests = errors.New("the report names more tests than a report may hold")

func ParseRunReport(repository GovernedRepository, head CommitSHA, results []TestResult) (RunReport, error) {
	if len(results) > MaxReportedTests {
		return RunReport{}, ErrTooManyReportedTests
	}
	return RunReport{Repository: repository, Head: head, Results: results}, nil
}

func TestsConclusionOf(plan TestPlan, results []TestResult) Conclusion {
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
	for _, suite := range plan.Suites {
		if !reported[suite] {
			return ConclusionFailure
		}
	}
	return ConclusionSuccess
}
