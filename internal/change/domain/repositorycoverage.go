// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type RepositoryCoverage struct {
	suites map[SuiteName]SuiteCoverage
}

func (repositoryCoverage RepositoryCoverage) Replacing(suite SuiteName, coverage SuiteCoverage) RepositoryCoverage {
	return RepositoryCoverage{suites: map[SuiteName]SuiteCoverage{suite: coverage}}
}

func (repositoryCoverage RepositoryCoverage) SuiteCoverageOf(suite SuiteName) (SuiteCoverage, error) {
	return repositoryCoverage.suites[suite], nil
}
