// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "slices"

type ReportedCoverage struct {
	paths []RepositoryPath
	tests []ReportedTest
}

func NewReportedCoverage(paths []RepositoryPath, tests []ReportedTest) ReportedCoverage {
	every := slices.Clone(paths)
	for _, test := range tests {
		for _, file := range test.files {
			every = append(every, file.path)
		}
	}
	ordered := slices.SortedFunc(slices.Values(every), RepositoryPath.compare)
	return ReportedCoverage{paths: slices.Compact(ordered), tests: slices.Clone(tests)}
}
