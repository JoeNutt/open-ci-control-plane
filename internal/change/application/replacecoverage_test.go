// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/memory"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	uploadingRepository = "JoeNutt/open-ci-control-plane"
	uploadedSuite       = "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	coveredFile         = "internal/change/domain/effect.go"
)

func TestReplaceCoverageStoresTheSuitesMap(t *testing.T) {
	maps := memory.NewCoverageMaps(domain.DeclaredCoverageLimits())
	repository, suite, path := parsedRepository(t), parsedSuite(t), parsedPath(t)

	useCase := application.ReplaceCoverage{Maps: maps, Paths: repositoryHolding{path}}
	if err := useCase.Execute(t.Context(), repository, suite, reportOf(path)); err != nil {
		t.Fatalf("replace the suite's coverage: %v", err)
	}

	served, err := maps.SuiteCoverageOf(t.Context(), repository, suite)
	if err != nil {
		t.Fatalf("serve the replaced map: %v", err)
	}
	if !served.Covers(path) {
		t.Fatalf("the served map does not cover %s", path)
	}
}

func TestReplaceCoverageRefusesAPathTheRepositoryDoesNotHoldAndStoresNothing(t *testing.T) {
	maps := memory.NewCoverageMaps(domain.DeclaredCoverageLimits())
	repository, suite, path := parsedRepository(t), parsedSuite(t), parsedPath(t)

	useCase := application.ReplaceCoverage{Maps: maps, Paths: repositoryHolding{}}
	err := useCase.Execute(t.Context(), repository, suite, reportOf(path))

	if !errors.Is(err, domain.ErrPathOutsideRepository) {
		t.Fatalf("want ErrPathOutsideRepository, got %v", err)
	}
	if _, err := maps.SuiteCoverageOf(t.Context(), repository, suite); !errors.Is(err, domain.ErrNoCoverageMap) {
		t.Fatalf("a refused report was stored: %v", err)
	}
}

func TestReplaceCoverageFailsWhenTheRepositorysPathsCannotBeRead(t *testing.T) {
	maps := memory.NewCoverageMaps(domain.DeclaredCoverageLimits())
	repository, suite := parsedRepository(t), parsedSuite(t)
	unreadable := errors.New("the forge is unreachable")

	useCase := application.ReplaceCoverage{Maps: maps, Paths: unreadablePaths{unreadable}}
	err := useCase.Execute(t.Context(), repository, suite, reportOf(parsedPath(t)))

	if !errors.Is(err, unreadable) {
		t.Fatalf("want the read failure, got %v", err)
	}
	if _, err := maps.SuiteCoverageOf(t.Context(), repository, suite); !errors.Is(err, domain.ErrNoCoverageMap) {
		t.Fatalf("an upload whose paths were never read was stored: %v", err)
	}
}

type unreadablePaths struct{ failure error }

func (unreadable unreadablePaths) KnownPathsOf(context.Context, domain.Repository) (domain.KnownPaths, error) {
	return domain.KnownPaths{}, unreadable.failure
}

type repositoryHolding []domain.RepositoryPath

func (held repositoryHolding) KnownPathsOf(context.Context, domain.Repository) (domain.KnownPaths, error) {
	return domain.NewKnownPaths(held), nil
}

func reportOf(path domain.RepositoryPath) []domain.ReportedFile {
	return []domain.ReportedFile{domain.NewReportedFile(path, true)}
}

func parsedRepository(t *testing.T) domain.Repository {
	t.Helper()

	repository, err := domain.ParseRepository(uploadingRepository)
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}
	return repository
}

func parsedSuite(t *testing.T) domain.SuiteName {
	t.Helper()

	suite, err := domain.ParseSuiteName(uploadedSuite)
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	return suite
}

func parsedPath(t *testing.T) domain.RepositoryPath {
	t.Helper()

	path, err := domain.ParseRepositoryPath(coveredFile)
	if err != nil {
		t.Fatalf("parse path: %v", err)
	}
	return path
}
