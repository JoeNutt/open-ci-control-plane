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
	Rate    *UploadRate
}

type CoverageUpload struct {
	Token      domain.IdentityToken
	Repository domain.GovernedRepository
	Suite      domain.CoverageSuite
	Report     domain.CoverageReport
}

type ReplacedCoverage struct {
	Suite          domain.CoverageSuite
	FileEntryCount int
}

func (useCase ReplaceCoverage) Execute(ctx context.Context, upload CoverageUpload) (ReplacedCoverage, error) {
	if err := useCase.authorise(ctx, upload); err != nil {
		return ReplacedCoverage{}, err
	}
	if err := useCase.Rate.Admit(upload.Repository); err != nil {
		return ReplacedCoverage{}, fmt.Errorf("admit coverage for %s: %w", upload.Repository, err)
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
	if err := callerIsRepository(ctx, useCase.Callers, upload.Token, upload.Repository); err != nil {
		return fmt.Errorf("authorise coverage for %s: %w", upload.Repository, err)
	}
	return nil
}

func callerIsRepository(
	ctx context.Context, callers domain.CallerRepositories, token domain.IdentityToken, repository domain.GovernedRepository,
) error {
	caller, err := callers.RepositoryOf(ctx, token)
	if err != nil {
		return fmt.Errorf("verify the caller: %w", err)
	}
	if caller != repository {
		return fmt.Errorf("%w: issued to %s, presented for %s", ErrForeignRepository, caller, repository)
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
