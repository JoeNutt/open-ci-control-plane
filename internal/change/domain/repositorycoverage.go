// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"maps"
)

var ErrNoCoverageMap = errors.New("no coverage map is held for the suite")

type RepositoryCoverage struct {
	suites map[GoImportPath]SuiteCoverage
}

func (repositoryCoverage RepositoryCoverage) Replacing(
	suite GoImportPath, coverage SuiteCoverage, limits CoverageLimits,
) (RepositoryCoverage, error) {
	if breach := limits.SuitesBreach(repositoryCoverage.suiteCountWith(suite)); breach != nil {
		return RepositoryCoverage{}, breach
	}
	suites := make(map[GoImportPath]SuiteCoverage, len(repositoryCoverage.suites)+1)
	maps.Copy(suites, repositoryCoverage.suites)
	suites[suite] = coverage
	return RepositoryCoverage{suites: suites}, nil
}

func (repositoryCoverage RepositoryCoverage) suiteCountWith(suite GoImportPath) int {
	if _, isHeld := repositoryCoverage.suites[suite]; isHeld {
		return len(repositoryCoverage.suites)
	}
	return len(repositoryCoverage.suites) + 1
}

func (repositoryCoverage RepositoryCoverage) SuiteCoverageOf(suite GoImportPath) (SuiteCoverage, error) {
	coverage, isHeld := repositoryCoverage.suites[suite]
	if !isHeld {
		return SuiteCoverage{}, ErrNoCoverageMap
	}
	return coverage, nil
}
