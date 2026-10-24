// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"iter"
	"slices"
)

var errUnknownPath = fmt.Errorf("%w: the repository holds no such path", ErrPathOutsideRepository)

type SuiteCoverage struct {
	tests []ReportedTest
}

func ParseSuiteCoverage(reported []ReportedTest, known KnownPaths) (SuiteCoverage, error) {
	coverage := SuiteCoverage{tests: slices.Clone(reported)}
	for file := range coverage.testedFiles() {
		if !known.Contains(file.path) {
			return SuiteCoverage{}, errUnknownPath
		}
	}
	return coverage, nil
}

func (coverage SuiteCoverage) FileEntryCount() int {
	paths := map[RepositoryPath]bool{}
	for file := range coverage.testedFiles() {
		paths[file.path] = true
	}
	return len(paths)
}

func (coverage SuiteCoverage) Covers(path RepositoryPath) bool {
	for file := range coverage.testedFiles() {
		if file.path == path && file.isExercised() {
			return true
		}
	}
	return false
}

func (coverage SuiteCoverage) testedFiles() iter.Seq[TestedFile] {
	return func(yield func(TestedFile) bool) {
		for _, test := range coverage.tests {
			for _, file := range test.files {
				if !yield(file) {
					return
				}
			}
		}
	}
}
