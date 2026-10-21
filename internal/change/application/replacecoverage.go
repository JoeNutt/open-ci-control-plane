// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrForeignRepository = errors.New("the token was issued to another repository")

type ReplaceCoverage struct {
	Maps    domain.CoverageMaps
	Paths   domain.RepositoryPaths
	Callers domain.CallerRepositories
}

type CoverageUpload struct {
	Token      domain.IdentityToken
	Repository domain.Repository
	Suite      domain.SuiteName
	Reported   []domain.ReportedFile
}

func (useCase ReplaceCoverage) Execute(ctx context.Context, upload CoverageUpload) error {
	caller, _ := useCase.Callers.RepositoryOf(ctx, upload.Token)
	if caller != upload.Repository {
		return fmt.Errorf("%w: issued to %s, uploading for %s", ErrForeignRepository, caller, upload.Repository)
	}
	known, err := useCase.Paths.KnownPathsOf(ctx, upload.Repository)
	if err != nil {
		return fmt.Errorf("read the paths %s holds: %w", upload.Repository, err)
	}
	coverage, err := domain.ParseSuiteCoverage(upload.Reported, known)
	if err != nil {
		return fmt.Errorf("refuse coverage of %s for %s: %w", upload.Suite, upload.Repository, err)
	}
	return useCase.Maps.ReplaceSuiteCoverage(ctx, upload.Repository, upload.Suite, coverage)
}
