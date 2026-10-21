// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type ReplaceCoverage struct {
	Maps    domain.CoverageMaps
	Paths   domain.RepositoryPaths
	Callers domain.CallerRepositories
}

var ErrForeignRepository = errors.New("the token was issued to another repository")

func (useCase ReplaceCoverage) Execute(
	ctx context.Context, token domain.IdentityToken, repository domain.Repository, suite domain.SuiteName, reported []domain.ReportedFile,
) error {
	caller, _ := useCase.Callers.RepositoryOf(ctx, token)
	if caller != repository {
		return ErrForeignRepository
	}
	known, err := useCase.Paths.KnownPathsOf(ctx, repository)
	if err != nil {
		return fmt.Errorf("read the paths %s holds: %w", repository, err)
	}
	coverage, err := domain.ParseSuiteCoverage(reported, known)
	if err != nil {
		return fmt.Errorf("refuse coverage of %s for %s: %w", suite, repository, err)
	}
	return useCase.Maps.ReplaceSuiteCoverage(ctx, repository, suite, coverage)
}
