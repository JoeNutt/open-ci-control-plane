// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
)

var errUnknownPath = fmt.Errorf("%w: the repository holds no such path", ErrPathOutsideRepository)

type SuiteCoverage struct {
	fileEntryCount int
	reported       []ReportedFile
}

func ParseSuiteCoverage(reported []ReportedFile, known KnownPaths) (SuiteCoverage, error) {
	isUnknown := func(file ReportedFile) bool { return !known.Contains(file.Path()) }
	if slices.ContainsFunc(reported, isUnknown) {
		return SuiteCoverage{}, errUnknownPath
	}
	return SuiteCoverage{fileEntryCount: len(reported), reported: reported}, nil
}

func (coverage SuiteCoverage) Covers(path RepositoryPath) bool {
	for _, file := range coverage.reported {
		if file.Path() == path && file.IsExercised() {
			return true
		}
	}
	return false
}

func (coverage SuiteCoverage) FileEntryCount() int { return coverage.fileEntryCount }
