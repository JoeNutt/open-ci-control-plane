// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type ReplaceCoverage struct {
	Maps domain.CoverageMaps
}

func (useCase ReplaceCoverage) Execute(
	ctx context.Context, repository domain.Repository, suite domain.SuiteName, coverage domain.SuiteCoverage,
) error {
	return useCase.Maps.ReplaceSuiteCoverage(ctx, repository, suite, coverage)
}
