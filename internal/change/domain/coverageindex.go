// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"cmp"
	"errors"
	"slices"
	"strings"
)

var errNoTestAtIndex = errors.New("the coverage index numbers no test at that position")

type coveredTest struct {
	suite CoverageSuite
	name  TopLevelTestName
}

type coverageIndex struct {
	tests []coveredTest
}

func indexOf(suites map[CoverageSuite]SuiteCoverage) coverageIndex {
	return coverageIndex{tests: denselyNumberedTestsOf(suites)}
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

func (index coverageIndex) testAt(dense int) (coveredTest, error) {
	if dense < 0 || dense >= len(index.tests) {
		return coveredTest{}, errNoTestAtIndex
	}
	return index.tests[dense], nil
}
