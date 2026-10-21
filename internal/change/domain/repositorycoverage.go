// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"maps"
)

var ErrNoCoverageMap = errors.New("no coverage map is held for the suite")

type RepositoryCoverage struct {
	suites map[SuiteName]SuiteCoverage
}

func (repositoryCoverage RepositoryCoverage) Replacing(
	suite SuiteName, coverage SuiteCoverage, limits CoverageLimits,
) (RepositoryCoverage, error) {
	suites := make(map[SuiteName]SuiteCoverage, len(repositoryCoverage.suites)+1)
	maps.Copy(suites, repositoryCoverage.suites)
	suites[suite] = coverage
	if len(suites) > limits.MaxSuitesPerRepository {
		return RepositoryCoverage{}, ErrTooManySuites
	}
	return RepositoryCoverage{suites: suites}, nil
}

func (repositoryCoverage RepositoryCoverage) SuiteCoverageOf(suite SuiteName) (SuiteCoverage, error) {
	coverage, isHeld := repositoryCoverage.suites[suite]
	if !isHeld {
		return SuiteCoverage{}, ErrNoCoverageMap
	}
	return coverage, nil
}
