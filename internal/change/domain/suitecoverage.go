// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type SuiteCoverage struct{}

func ParseSuiteCoverage(reported []ReportedFile, known KnownPaths) (SuiteCoverage, error) {
	for _, file := range reported {
		if !known.Contains(file.Path()) {
			return SuiteCoverage{}, ErrPathOutsideRepository
		}
	}
	return SuiteCoverage{}, nil
}
