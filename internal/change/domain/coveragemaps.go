// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type CoverageMaps interface {
	ReplaceSuiteCoverage(ctx context.Context, repository GovernedRepository, suite CoverageSuite, coverage SuiteCoverage) error
	SuiteCoverageOf(ctx context.Context, repository GovernedRepository, suite CoverageSuite) (SuiteCoverage, error)
	RepositoryCoverageOf(ctx context.Context, repository GovernedRepository) (RepositoryCoverage, error)
}
