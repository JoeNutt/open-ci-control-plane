// SPDX-License-Identifier: GPL-3.0-or-later

package domain

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

func TestsConclusionOf(results []TestResult) Conclusion {
	for _, result := range results {
		if result.Outcome == TestOutcomeFail {
			return ConclusionFailure
		}
	}
	return ConclusionSuccess
}
