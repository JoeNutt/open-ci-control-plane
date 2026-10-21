// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type ReplaceCoverage struct {
	Maps  domain.CoverageMaps
	Paths domain.RepositoryPaths
}

func (useCase ReplaceCoverage) Execute(
	ctx context.Context, repository domain.Repository, suite domain.SuiteName, reported []domain.ReportedFile,
) error {
	known, _ := useCase.Paths.KnownPathsOf(ctx, repository)
	coverage, err := domain.ParseSuiteCoverage(reported, known)
	if err != nil {
		return err
	}
	return useCase.Maps.ReplaceSuiteCoverage(ctx, repository, suite, coverage)
}
