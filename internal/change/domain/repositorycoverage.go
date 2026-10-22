// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"maps"
	"slices"
	"strings"
)

var ErrNoCoverageMap = errors.New("no coverage map is held for the suite")

type RepositoryCoverage struct {
	suites map[CoverageSuite]SuiteCoverage
}

func (repositoryCoverage RepositoryCoverage) Replacing(
	suite CoverageSuite, coverage SuiteCoverage, limits CoverageLimits,
) (RepositoryCoverage, error) {
	if breach := limits.SuitesBreach(repositoryCoverage.suiteCountWith(suite)); breach != nil {
		return RepositoryCoverage{}, breach
	}
	suites := make(map[CoverageSuite]SuiteCoverage, len(repositoryCoverage.suites)+1)
	maps.Copy(suites, repositoryCoverage.suites)
	suites[suite] = coverage
	return RepositoryCoverage{suites: suites}, nil
}

func (repositoryCoverage RepositoryCoverage) suiteCountWith(suite CoverageSuite) int {
	if _, isHeld := repositoryCoverage.suites[suite]; isHeld {
		return len(repositoryCoverage.suites)
	}
	return len(repositoryCoverage.suites) + 1
}

func (repositoryCoverage RepositoryCoverage) SuiteCoverageOf(suite CoverageSuite) (SuiteCoverage, error) {
	coverage, isHeld := repositoryCoverage.suites[suite]
	if !isHeld {
		return SuiteCoverage{}, ErrNoCoverageMap
	}
	return coverage, nil
}

func (repositoryCoverage RepositoryCoverage) suitesCoveringAny(files []File) ([]CoverageSuite, error) {
	var covering []CoverageSuite
	for _, file := range files {
		path, err := ParseRepositoryPath(file.Path)
		if err != nil {
			return nil, err
		}
		covering = append(covering, repositoryCoverage.suitesCovering(path)...)
	}
	slices.SortFunc(covering, func(one, other CoverageSuite) int { return strings.Compare(one.String(), other.String()) })
	return slices.Compact(covering), nil
}

func (repositoryCoverage RepositoryCoverage) suitesCovering(path RepositoryPath) []CoverageSuite {
	var covering []CoverageSuite
	for suite, coverage := range repositoryCoverage.suites {
		if coverage.Covers(path) {
			covering = append(covering, suite)
		}
	}
	return covering
}
