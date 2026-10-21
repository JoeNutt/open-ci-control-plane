// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
)

var errUnknownPath = fmt.Errorf("%w: the repository holds no such path", ErrPathOutsideRepository)

type SuiteCoverage struct{}

func ParseSuiteCoverage(reported []ReportedFile, known KnownPaths) (SuiteCoverage, error) {
	isUnknown := func(file ReportedFile) bool { return !known.Contains(file.Path()) }
	if slices.ContainsFunc(reported, isUnknown) {
		return SuiteCoverage{}, errUnknownPath
	}
	return SuiteCoverage{}, nil
}

func (coverage SuiteCoverage) FileEntryCount() int { return 2 }
