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
	revision      *indexRevision
	tests         []coveredTest
	testsOfSuites map[CoverageSuite]testSpan
	postings      coveragePostings
}

type suiteTest struct {
	suite CoverageSuite
	test  ReportedTest
}

func indexOf(suites map[CoverageSuite]SuiteCoverage) coverageIndex {
	numbered := denselyNumberedTestsOf(suites)
	return coverageIndex{
		revision:      &indexRevision{testCount: len(numbered)},
		tests:         coveredTestsOf(numbered),
		testsOfSuites: testSpansOf(numbered),
		postings:      postingsOf(edgesOf(numbered)),
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

type testSpan struct {
	first int
	end   int
}

func (span testSpan) testCount() int { return span.end - span.first }

func testSpansOf(numbered []suiteTest) map[CoverageSuite]testSpan {
	spans := map[CoverageSuite]testSpan{}
	for dense, each := range numbered {
		span, isHeld := spans[each.suite]
		if !isHeld {
			span.first = dense
		}
		span.end = dense + 1
		spans[each.suite] = span
	}
	return spans
}

func (index coverageIndex) testAt(dense int) (coveredTest, error) {
	if !index.numbers(dense) {
		return coveredTest{}, fmt.Errorf("%w: position %d of %d tests", errNoTestAtIndex, dense, len(index.tests))
	}
	return index.tests[dense], nil
}

func (index coverageIndex) testsOfSuite(suite CoverageSuite) (testSet, error) {
	span, isIndexed := index.testsOfSuites[suite]
	if !isIndexed {
		return testSet{}, fmt.Errorf("%w: %s", errUnindexedSuite, suite)
	}
	members := make([]uint32, 0, span.testCount())
	for dense := span.first; dense < span.end; dense++ {
		members = append(members, uint32(dense))
	}
	return testSetOf(index.revision, members), nil
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

func (index coverageIndex) holdsAnyTest() bool { return len(index.tests) > 0 }

func (index coverageIndex) numbers(dense int) bool {
	return 0 <= dense && dense < len(index.tests)
}
