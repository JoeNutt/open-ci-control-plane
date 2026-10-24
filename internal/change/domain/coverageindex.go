// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"cmp"
	"errors"
	"fmt"
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
	if !index.numbers(dense) {
		return coveredTest{}, fmt.Errorf("%w: position %d of %d tests", errNoTestAtIndex, dense, len(index.tests))
	}
	return index.tests[dense], nil
}

func (index coverageIndex) testCountOf(suite CoverageSuite) (int, error) {
	count := 0
	for _, test := range index.tests {
		if test.suite == suite {
			count++
		}
	}
	return count, nil
}

func (index coverageIndex) numbers(dense int) bool {
	return 0 <= dense && dense < len(index.tests)
}
