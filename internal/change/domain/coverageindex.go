// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

var (
	errNoTestAtIndex  = errors.New("the coverage index numbers no test at that position")
	errUnindexedSuite = errors.New("the coverage index holds no coverage for the suite")
	errUnindexedFile  = errors.New("the coverage index has never seen a test run the file")
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

type suiteTest struct {
	suite CoverageSuite
	test  ReportedTest
}

func indexOf(suites map[CoverageSuite]SuiteCoverage) coverageIndex {
	numbered := denselyNumberedTestsOf(suites)
	return coverageIndex{
		revision:         &indexRevision{testCount: len(numbered)},
		tests:            coveredTestsOf(numbered),
		testCountBySuite: testCountBySuiteOf(suites),
		postings:         postingsOf(edgesOf(numbered)),
	}
}

func denselyNumberedTestsOf(suites map[CoverageSuite]SuiteCoverage) []suiteTest {
	var numbered []suiteTest
	for suite, coverage := range suites {
		for _, test := range coverage.tests {
			numbered = append(numbered, suiteTest{suite: suite, test: test})
		}
	}
	slices.SortFunc(numbered, suiteThenNameOrder)
	return numbered
}

func suiteThenNameOrder(one, other suiteTest) int {
	return cmp.Or(
		one.suite.compare(other.suite),
		one.test.name.compare(other.test.name),
	)
}

func coveredTestsOf(numbered []suiteTest) []coveredTest {
	tests := make([]coveredTest, 0, len(numbered))
	for _, each := range numbered {
		tests = append(tests, coveredTest{suite: each.suite, name: each.test.name})
	}
	return tests
}

func testCountBySuiteOf(suites map[CoverageSuite]SuiteCoverage) map[CoverageSuite]int {
	counts := make(map[CoverageSuite]int, len(suites))
	for suite, coverage := range suites {
		counts[suite] = len(coverage.tests)
	}
	return counts
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
	items, err := index.itemsOf(path)
	if err != nil {
		return testSet{}, err
	}
	return testSetOf(index.revision, index.postings.testsOn(items)), nil
}

func (index coverageIndex) testsCovering(path RepositoryPath, lines []uint32) (testSet, error) {
	items, err := index.itemsOf(path)
	if err != nil {
		return testSet{}, err
	}
	return testSetOf(index.revision, index.postings.testsOnLines(items, lines)), nil
}

func (index coverageIndex) itemsOf(path RepositoryPath) (itemSpan, error) {
	items, isIndexed := index.postings.itemsOfFile[path]
	if !isIndexed {
		return itemSpan{}, fmt.Errorf("%w: %s", errUnindexedFile, path)
	}
	return items, nil
}

func (index coverageIndex) numbers(dense int) bool {
	return 0 <= dense && dense < len(index.tests)
}
