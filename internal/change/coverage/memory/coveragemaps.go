// SPDX-License-Identifier: GPL-3.0-or-later

package memory

import "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"

type CoverageMaps struct {
	held   map[domain.Repository]domain.RepositoryCoverage
	limits domain.CoverageLimits
}

func NewCoverageMaps(limits domain.CoverageLimits) *CoverageMaps {
	return &CoverageMaps{held: map[domain.Repository]domain.RepositoryCoverage{}, limits: limits}
}

func (maps *CoverageMaps) ReplaceSuiteCoverage(
	repository domain.Repository, suite domain.SuiteName, coverage domain.SuiteCoverage,
) error {
	replaced, err := maps.held[repository].Replacing(suite, coverage, maps.limits)
	if err != nil {
		return err
	}
	maps.held[repository] = replaced
	return nil
}

func (maps *CoverageMaps) SuiteCoverageOf(repository domain.Repository, suite domain.SuiteName) (domain.SuiteCoverage, error) {
	return maps.held[repository].SuiteCoverageOf(suite)
}
