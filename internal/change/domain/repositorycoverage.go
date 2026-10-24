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
	size   indexSize
	lazy   *lazyIndex
}

func (repositoryCoverage RepositoryCoverage) Replacing(
	suite CoverageSuite, coverage SuiteCoverage, limits CoverageLimits,
) (RepositoryCoverage, error) {
	if breach := limits.SuitesBreach(repositoryCoverage.suiteCountWith(suite)); breach != nil {
		return RepositoryCoverage{}, breach
	}
	size := repositoryCoverage.size.replacing(repositoryCoverage.suites[suite].size, coverage.size)
	if breach := size.breachIn(limits); breach != nil {
		return RepositoryCoverage{}, breach
	}
	suites := make(map[CoverageSuite]SuiteCoverage, len(repositoryCoverage.suites)+1)
	maps.Copy(suites, repositoryCoverage.suites)
	suites[suite] = coverage
	return RepositoryCoverage{suites: suites, size: size, lazy: &lazyIndex{suites: suites}}, nil
}

func (repositoryCoverage RepositoryCoverage) coverageIndex() coverageIndex {
	return repositoryCoverage.lazy.index()
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

type coverageSelection struct {
	suites            []CoverageSuite
	isEveryFileMapped bool
	isAnyMapHeld      bool
}

func (repositoryCoverage RepositoryCoverage) selectionFor(files []File) (coverageSelection, error) {
	selection := coverageSelection{isEveryFileMapped: true, isAnyMapHeld: len(repositoryCoverage.suites) > 0}
	for _, file := range files {
		path, err := ParseRepositoryPath(file.Path)
		if err != nil {
			return coverageSelection{}, err
		}
		covering := repositoryCoverage.suitesCovering(path)
		selection.isEveryFileMapped = selection.isEveryFileMapped && len(covering) > 0
		selection.suites = append(selection.suites, covering...)
	}
	slices.SortFunc(selection.suites, func(one, other CoverageSuite) int { return strings.Compare(one.String(), other.String()) })
	selection.suites = slices.Compact(selection.suites)
	return selection, nil
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
