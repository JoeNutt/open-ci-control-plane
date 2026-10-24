// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
)

var (
	errNoTestAtIndex  = errors.New("the coverage index numbers no test at that position")
	errUnindexedSuite = errors.New("the coverage index holds no coverage for the suite")
)

type coveredTest struct {
	suite CoverageSuite
	name  TopLevelTestName
}

type coverageIndex struct {
	revision         *indexRevision
	tests            []coveredTest
	testCountBySuite map[CoverageSuite]int
	postings         coveragePostings
}

func indexOf(suites map[CoverageSuite]SuiteCoverage) coverageIndex {
	tests := denselyNumberedTestsOf(suites)
	return coverageIndex{
		revision:         &indexRevision{testCount: len(tests)},
		tests:            tests,
		testCountBySuite: testCountBySuiteOf(suites),
		postings:         postingsOf(edgesOf(suites, positionsOf(tests))),
	}
}

func denselyNumberedTestsOf(suites map[CoverageSuite]SuiteCoverage) []coveredTest {
	var tests []coveredTest
	for suite, coverage := range suites {
		for _, test := range coverage.tests {
			tests = append(tests, coveredTest{suite: suite, name: test.name})
		}
	}
	slices.SortFunc(tests, suiteThenNameOrder)
	return tests
}

func suiteThenNameOrder(one, other coveredTest) int {
	return cmp.Or(
		strings.Compare(one.suite.String(), other.suite.String()),
		strings.Compare(one.name.name, other.name.name),
	)
}

func testCountBySuiteOf(suites map[CoverageSuite]SuiteCoverage) map[CoverageSuite]int {
	counts := make(map[CoverageSuite]int, len(suites))
	for suite, coverage := range suites {
		counts[suite] = len(coverage.tests)
	}
	return counts
}

func positionsOf(tests []coveredTest) map[coveredTest]uint32 {
	positions := make(map[coveredTest]uint32, len(tests))
	for position, test := range tests {
		positions[test] = uint32(position)
	}
	return positions
}

func (index coverageIndex) testAt(dense int) (coveredTest, error) {
	if !index.numbers(dense) {
		return coveredTest{}, fmt.Errorf("%w: position %d of %d tests", errNoTestAtIndex, dense, len(index.tests))
	}
	return index.tests[dense], nil
}

func (index coverageIndex) testCountOf(suite CoverageSuite) (int, error) {
	count, isIndexed := index.testCountBySuite[suite]
	if !isIndexed {
		return 0, fmt.Errorf("%w: %s", errUnindexedSuite, suite)
	}
	return count, nil
}

func (index coverageIndex) testsReaching(path RepositoryPath) (testSet, error) {
	return testSetOf(index.revision, index.postings.testsReaching(path)), nil
}

func (index coverageIndex) numbers(dense int) bool {
	return 0 <= dense && dense < len(index.tests)
}
