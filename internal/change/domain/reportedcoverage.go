// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"slices"
	"strings"
)

type ReportedCoverage struct {
	paths []RepositoryPath
	tests []ReportedTest
}

func NewReportedCoverage(paths []RepositoryPath, tests []ReportedTest) ReportedCoverage {
	ordered := slices.SortedFunc(slices.Values(paths), func(one, other RepositoryPath) int {
		return strings.Compare(one.String(), other.String())
	})
	return ReportedCoverage{paths: slices.Compact(ordered), tests: slices.Clone(tests)}
}
