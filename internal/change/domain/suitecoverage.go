// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
)

var errUnknownPath = fmt.Errorf("%w: the repository holds no such path", ErrPathOutsideRepository)

type SuiteCoverage struct {
	tests []ReportedTest
}

func ParseSuiteCoverage(reported []ReportedTest, known KnownPaths) (SuiteCoverage, error) {
	for _, test := range reported {
		for _, file := range test.files {
			if !known.Contains(file.path) {
				return SuiteCoverage{}, errUnknownPath
			}
		}
	}
	return SuiteCoverage{tests: slices.Clone(reported)}, nil
}

func (coverage SuiteCoverage) FileEntryCount() int {
	paths := map[RepositoryPath]bool{}
	for _, test := range coverage.tests {
		for _, file := range test.files {
			paths[file.path] = true
		}
	}
	return len(paths)
}

func (coverage SuiteCoverage) Covers(path RepositoryPath) bool {
	for _, test := range coverage.tests {
		for _, file := range test.files {
			if file.path == path && len(file.hitLines) > 0 {
				return true
			}
		}
	}
	return false
}
