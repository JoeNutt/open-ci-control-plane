// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type CoverageMaps interface {
	ReplaceSuiteCoverage(ctx context.Context, repository Repository, suite SuiteName, coverage SuiteCoverage) error
	SuiteCoverageOf(ctx context.Context, repository Repository, suite SuiteName) (SuiteCoverage, error)
}
