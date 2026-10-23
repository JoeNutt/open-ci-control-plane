// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

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
	repository GovernedRepository
	head       CommitSHA
	results    []TestResult
}

const MaxReportedTests = 10000

var ErrTooManyReportedTests = errors.New("the report names more tests than a report may hold")

func ParseRunReport(repository GovernedRepository, head CommitSHA, results []TestResult) (RunReport, error) {
	if len(results) > MaxReportedTests {
		return RunReport{}, fmt.Errorf("%w: %d tests, at most %d", ErrTooManyReportedTests, len(results), MaxReportedTests)
	}
	return RunReport{repository: repository, head: head, results: results}, nil
}

func (report RunReport) Repository() GovernedRepository { return report.repository }

func (report RunReport) Head() CommitSHA { return report.head }

func (report RunReport) Results() []TestResult { return slices.Clone(report.results) }

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
