// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"iter"
	"slices"
)

var errUnknownPath = fmt.Errorf("%w: the repository holds no such path", ErrPathOutsideRepository)

type SuiteCoverage struct {
	fileEntryCount int
	tests          []ReportedTest
}

func ParseSuiteCoverage(reported ReportedCoverage, known KnownPaths) (SuiteCoverage, error) {
	isUnknown := func(path RepositoryPath) bool { return !known.Contains(path) }
	if slices.ContainsFunc(reported.paths, isUnknown) {
		return SuiteCoverage{}, errUnknownPath
	}
	return SuiteCoverage{fileEntryCount: len(reported.paths), tests: reported.tests}, nil
}

func (coverage SuiteCoverage) FileEntryCount() int { return coverage.fileEntryCount }

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
