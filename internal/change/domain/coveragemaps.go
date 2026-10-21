// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type CoverageMaps interface {
	ReplaceSuiteCoverage(ctx context.Context, repository GovernedRepository, suite SuiteName, coverage SuiteCoverage) error
	SuiteCoverageOf(ctx context.Context, repository GovernedRepository, suite SuiteName) (SuiteCoverage, error)
}
