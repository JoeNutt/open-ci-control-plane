// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
)

var errUnknownPath = fmt.Errorf("%w: the repository holds no such path", ErrPathOutsideRepository)

type SuiteCoverage struct {
	reported []ReportedFile
}

func ParseSuiteCoverage(reported []ReportedFile, known KnownPaths) (SuiteCoverage, error) {
	isUnknown := func(file ReportedFile) bool { return !known.Contains(file.Path()) }
	if slices.ContainsFunc(reported, isUnknown) {
		return SuiteCoverage{}, errUnknownPath
	}
	return SuiteCoverage{reported: slices.Clone(reported)}, nil
}

func (coverage SuiteCoverage) FileEntryCount() int { return len(coverage.reported) }

func (coverage SuiteCoverage) Covers(path RepositoryPath) bool {
	exercises := func(file ReportedFile) bool { return file.IsExercised() && file.Path() == path }
	return slices.ContainsFunc(coverage.reported, exercises)
}
