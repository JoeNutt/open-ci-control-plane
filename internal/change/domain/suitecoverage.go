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
	size           indexSize
}

func ParseSuiteCoverage(reported ReportedCoverage, known KnownPaths) (SuiteCoverage, error) {
	isUnknown := func(path RepositoryPath) bool { return !known.Contains(path) }
	if slices.ContainsFunc(reported.paths, isUnknown) {
		return SuiteCoverage{}, errUnknownPath
	}
	coverage := SuiteCoverage{fileEntryCount: len(reported.paths), tests: reported.tests}
	coverage.size = coverage.indexSizeOf()
	return coverage, nil
}

func (coverage SuiteCoverage) FileEntryCount() int { return coverage.fileEntryCount }

func (coverage SuiteCoverage) indexSizeOf() indexSize {
	size := indexSize{tests: len(coverage.tests)}
	for file := range coverage.testedFiles() {
		size.edges += len(file.hitLines)
	}
	return size
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
