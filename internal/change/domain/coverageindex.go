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
	itemsOfFile      map[RepositoryPath][2]int
	offsets          []uint32
	postings         []uint32
}

type coverageEdge struct {
	path RepositoryPath
	line uint32
	test uint32
}

func indexOf(suites map[CoverageSuite]SuiteCoverage) coverageIndex {
	tests := denselyNumberedTestsOf(suites)
	index := coverageIndex{
		revision:         &indexRevision{testCount: len(tests)},
		tests:            tests,
		testCountBySuite: testCountBySuiteOf(suites),
		itemsOfFile:      map[RepositoryPath][2]int{},
	}
	positionOf := map[coveredTest]uint32{}
	for position, test := range tests {
		positionOf[test] = uint32(position)
	}
	var edges []coverageEdge
	for suite, coverage := range suites {
		for _, test := range coverage.tests {
			dense := positionOf[coveredTest{suite: suite, name: test.name}]
			for _, file := range test.files {
				for _, line := range file.hitLines {
					edges = append(edges, coverageEdge{path: file.path, line: line, test: dense})
				}
			}
		}
	}
	slices.SortFunc(edges, func(one, other coverageEdge) int {
		return cmp.Or(strings.Compare(one.path.String(), other.path.String()), cmp.Compare(one.line, other.line), cmp.Compare(one.test, other.test))
	})
	item := -1
	for position, edge := range edges {
		if position == 0 || edge.path != edges[position-1].path || edge.line != edges[position-1].line {
			item++
			index.offsets = append(index.offsets, uint32(position))
			span, isHeld := index.itemsOfFile[edge.path]
			if !isHeld {
				span[0] = item
			}
			span[1] = item + 1
			index.itemsOfFile[edge.path] = span
		}
		index.postings = append(index.postings, edge.test)
	}
	index.offsets = append(index.offsets, uint32(len(edges)))
	return index
}

func (index coverageIndex) testsReaching(path RepositoryPath) (testSet, error) {
	span := index.itemsOfFile[path]
	words := make([]uint64, (len(index.tests)+testsPerWord-1)/testsPerWord)
	for _, test := range index.postings[index.offsets[span[0]]:index.offsets[span[1]]] {
		words[test/testsPerWord] |= 1 << (test % testsPerWord)
	}
	return testSet{revision: index.revision, words: words}, nil
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

func (index coverageIndex) numbers(dense int) bool {
	return 0 <= dense && dense < len(index.tests)
}
