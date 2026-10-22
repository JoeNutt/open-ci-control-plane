// SPDX-License-Identifier: GPL-3.0-or-later

package domain

const TestsCheckRunName = "open-ci/tests"

type TestOutcome string

const TestOutcomePass TestOutcome = "pass"

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
