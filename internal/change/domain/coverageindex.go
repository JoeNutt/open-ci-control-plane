// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"slices"
	"strings"
)

type coveredTest struct {
	suite CoverageSuite
	name  TopLevelTestName
}

type coverageIndex struct {
	tests []coveredTest
}

func indexOf(suites map[CoverageSuite]SuiteCoverage) coverageIndex {
	var tests []coveredTest
	for suite, coverage := range suites {
		for _, test := range coverage.tests {
			tests = append(tests, coveredTest{suite: suite, name: test.name})
		}
	}
	slices.SortFunc(tests, func(one, other coveredTest) int {
		if c := strings.Compare(one.suite.String(), other.suite.String()); c != 0 {
			return c
		}
		return strings.Compare(one.name.name, other.name.name)
	})
	return coverageIndex{tests: tests}
}

func (index coverageIndex) testAt(dense int) (coveredTest, error) {
	if dense >= len(index.tests) {
		return coveredTest{}, errors.New("no test")
	}
	return index.tests[dense], nil
}
