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
	Report     domain.CoverageReport
}

type ReplacedCoverage struct {
	Suite          domain.SuiteName
	FileEntryCount int
}

func (useCase ReplaceCoverage) Execute(ctx context.Context, upload CoverageUpload) (ReplacedCoverage, error) {
	if err := useCase.authorise(ctx, upload); err != nil {
		return ReplacedCoverage{}, err
	}
	coverage, err := useCase.coverageIn(ctx, upload)
	if err != nil {
		return ReplacedCoverage{}, fmt.Errorf("refuse coverage of %s for %s: %w", upload.Suite, upload.Repository, err)
	}
	if err := useCase.Maps.ReplaceSuiteCoverage(ctx, upload.Repository, upload.Suite, coverage); err != nil {
		return ReplacedCoverage{}, err
	}
	return ReplacedCoverage{Suite: upload.Suite, FileEntryCount: coverage.FileEntryCount()}, nil
}

func (useCase ReplaceCoverage) authorise(ctx context.Context, upload CoverageUpload) error {
	caller, err := useCase.Callers.RepositoryOf(ctx, upload.Token)
	if err != nil {
		return fmt.Errorf("verify who uploads coverage for %s: %w", upload.Repository, err)
	}
	if caller != upload.Repository {
		return fmt.Errorf("%w: issued to %s, uploading for %s", ErrForeignRepository, caller, upload.Repository)
	}
	return nil
}

func (useCase ReplaceCoverage) coverageIn(ctx context.Context, upload CoverageUpload) (domain.SuiteCoverage, error) {
	reported, err := upload.Report.ReportedFiles(domain.DeclaredCoverageLimits())
	if err != nil {
		return domain.SuiteCoverage{}, fmt.Errorf("read the artifact: %w", err)
	}
	known, err := useCase.Paths.KnownPathsOf(ctx, upload.Repository)
	if err != nil {
		return domain.SuiteCoverage{}, fmt.Errorf("read the paths it holds: %w", err)
	}
	return domain.ParseSuiteCoverage(reported, known)
}
