// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

const TestsCheckRunName = "open-ci/tests"

type TestOutcome string

const (
	TestOutcomePass TestOutcome = "pass"
	TestOutcomeFail TestOutcome = "fail"
	TestOutcomeSkip TestOutcome = "skip"
)

type TestResult struct {
	Suite   CoverageSuite
	Name    string
	Outcome TestOutcome
}

var reportableOutcomes = []TestOutcome{TestOutcomePass, TestOutcomeFail, TestOutcomeSkip}

var ErrMalformedTestResult = errors.New("the reported test is not one this service accepts")

func ParseTestResult(suite, name, outcome string) (TestResult, error) {
	if !slices.Contains(reportableOutcomes, TestOutcome(outcome)) {
		return TestResult{}, fmt.Errorf("%w: outcome %q", ErrMalformedTestResult, outcome)
	}
	if !isGoIdentifier(name) {
		return TestResult{}, fmt.Errorf("%w: name %q", ErrMalformedTestResult, name)
	}
	parsedSuite, err := ParseCoverageSuite(suite)
	if err != nil {
		return TestResult{}, fmt.Errorf("%w: %w", ErrMalformedTestResult, err)
	}
	return TestResult{Suite: parsedSuite, Name: name, Outcome: TestOutcome(outcome)}, nil
}

const (
	identifierLetters    = "_abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	identifierCharacters = identifierLetters + "0123456789"
)

func isGoIdentifier(name string) bool {
	return name != "" && strings.IndexByte(identifierLetters, name[0]) >= 0 && strings.Trim(name, identifierCharacters) == ""
}

type RunReport struct {
	repository GovernedRepository
	head       CommitSHA
	decisionID DecisionID
	results    []TestResult
}

const MaxReportedTests = 10000

var ErrTooManyReportedTests = errors.New("the report names more tests than a report may hold")

func ParseRunReport(repository GovernedRepository, head CommitSHA, decisionID DecisionID, results []TestResult) (RunReport, error) {
	if len(results) > MaxReportedTests {
		return RunReport{}, fmt.Errorf("%w: %d tests, at most %d", ErrTooManyReportedTests, len(results), MaxReportedTests)
	}
	return RunReport{repository: repository, head: head, decisionID: decisionID, results: results}, nil
}

func (report RunReport) Repository() GovernedRepository { return report.repository }

func (report RunReport) Head() CommitSHA { return report.head }

func (report RunReport) DecisionID() DecisionID { return report.decisionID }

func (report RunReport) Results() []TestResult { return slices.Clone(report.results) }
