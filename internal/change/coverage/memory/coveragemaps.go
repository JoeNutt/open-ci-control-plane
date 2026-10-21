// SPDX-License-Identifier: GPL-3.0-or-later

package memory

import (
	"context"
	"sync"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type CoverageMaps struct {
	guard  sync.RWMutex
	held   map[domain.Repository]domain.RepositoryCoverage
	limits domain.CoverageLimits
}

func NewCoverageMaps(limits domain.CoverageLimits) *CoverageMaps {
	return &CoverageMaps{held: map[domain.Repository]domain.RepositoryCoverage{}, limits: limits}
}

func (store *CoverageMaps) ReplaceSuiteCoverage(
	_ context.Context, repository domain.Repository, suite domain.SuiteName, coverage domain.SuiteCoverage,
) error {
	store.guard.Lock()
	defer store.guard.Unlock()

	replaced, err := store.held[repository].Replacing(suite, coverage, store.limits)
	if err != nil {
		return err
	}
	store.held[repository] = replaced
	return nil
}

func (store *CoverageMaps) SuiteCoverageOf(
	_ context.Context, repository domain.Repository, suite domain.SuiteName,
) (domain.SuiteCoverage, error) {
	store.guard.RLock()
	defer store.guard.RUnlock()

	return store.held[repository].SuiteCoverageOf(suite)
}
