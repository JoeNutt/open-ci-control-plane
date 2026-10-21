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
	strangerRepository  = "JoeNutt/open-ci-policy"
	uploadedSuite       = "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	coveredFile         = "internal/change/domain/effect.go"
)

func TestReplaceCoverageStoresTheSuitesMap(t *testing.T) {
	maps := memory.NewCoverageMaps(domain.DeclaredCoverageLimits())
	repository, suite, path := parsedRepository(t), parsedSuite(t), parsedPath(t)

	useCase := application.ReplaceCoverage{
		Maps: maps, Paths: repositoryHolding{path}, Callers: tokensNaming{repository}}
	if err := useCase.Execute(t.Context(), application.CoverageUpload{
		Token: presentedToken(t), Repository: repository, Suite: suite, Report: reportOf(path)}); err != nil {
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

	useCase := application.ReplaceCoverage{Maps: maps, Paths: repositoryHolding{}, Callers: tokensNaming{repository}}
	err := useCase.Execute(t.Context(), application.CoverageUpload{
		Token: presentedToken(t), Repository: repository, Suite: suite, Report: reportOf(path)})

	if !errors.Is(err, domain.ErrPathOutsideRepository) {
		t.Fatalf("want ErrPathOutsideRepository, got %v", err)
	}
	assertNothingStored(t, maps, repository, suite)
}

func TestReplaceCoverageFailsWhenTheRepositorysPathsCannotBeRead(t *testing.T) {
	maps := memory.NewCoverageMaps(domain.DeclaredCoverageLimits())
	repository, suite := parsedRepository(t), parsedSuite(t)
	unreadable := errors.New("the forge is unreachable")

	useCase := application.ReplaceCoverage{
		Maps: maps, Paths: unreadablePaths{unreadable}, Callers: tokensNaming{repository}}
	err := useCase.Execute(t.Context(), application.CoverageUpload{
		Token: presentedToken(t), Repository: repository, Suite: suite, Report: reportOf(parsedPath(t))})

	if !errors.Is(err, unreadable) {
		t.Fatalf("want the read failure, got %v", err)
	}
	assertNothingStored(t, maps, repository, suite)
}

func TestReplaceCoverageRefusesATokenFromAnotherRepositoryAndStoresNothing(t *testing.T) {
	maps := memory.NewCoverageMaps(domain.DeclaredCoverageLimits())
	repository, suite, path := parsedRepository(t), parsedSuite(t), parsedPath(t)
	stranger, err := domain.ParseRepository(strangerRepository)
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}

	useCase := application.ReplaceCoverage{
		Maps: maps, Paths: repositoryHolding{path}, Callers: tokensNaming{stranger}}
	err = useCase.Execute(t.Context(), application.CoverageUpload{
		Token: presentedToken(t), Repository: repository, Suite: suite, Report: reportOf(path)})

	if !errors.Is(err, application.ErrForeignRepository) {
		t.Fatalf("want ErrForeignRepository, got %v", err)
	}
	assertNothingStored(t, maps, repository, suite)
}

func TestReplaceCoverageFailsWithTheVerifiersRefusalAndStoresNothing(t *testing.T) {
	maps := memory.NewCoverageMaps(domain.DeclaredCoverageLimits())
	repository, suite, path := parsedRepository(t), parsedSuite(t), parsedPath(t)
	unverified := errors.New("token is absent, expired or not verifiable")

	useCase := application.ReplaceCoverage{
		Maps: maps, Paths: repositoryHolding{path}, Callers: unverifiedTokens{unverified}}
	err := useCase.Execute(t.Context(), application.CoverageUpload{
		Token: presentedToken(t), Repository: repository, Suite: suite, Report: reportOf(path)})

	if !errors.Is(err, unverified) || errors.Is(err, application.ErrForeignRepository) {
		t.Fatalf("want only the verifier's refusal, got %v", err)
	}
	assertNothingStored(t, maps, repository, suite)
}

func TestReplaceCoverageNeverReadsTheArtifactOfARefusedUploader(t *testing.T) {
	repository, suite, path := parsedRepository(t), parsedSuite(t), parsedPath(t)
	stranger, err := domain.ParseRepository(strangerRepository)
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}
	artifact := reportOf(path)

	useCase := application.ReplaceCoverage{Maps: memory.NewCoverageMaps(domain.DeclaredCoverageLimits()),
		Paths: repositoryHolding{path}, Callers: tokensNaming{stranger}}
	err = useCase.Execute(t.Context(), application.CoverageUpload{
		Token: presentedToken(t), Repository: repository, Suite: suite, Report: artifact})

	if !errors.Is(err, application.ErrForeignRepository) {
		t.Fatalf("want ErrForeignRepository, got %v", err)
	}
	if *artifact.reads != 0 {
		t.Fatalf("read a refused uploader's artifact %d times", *artifact.reads)
	}
}

type unverifiedTokens struct{ refusal error }

func (unverified unverifiedTokens) RepositoryOf(context.Context, domain.IdentityToken) (domain.Repository, error) {
	return domain.Repository{}, unverified.refusal
}

type tokensNaming struct{ repository domain.Repository }

func (naming tokensNaming) RepositoryOf(context.Context, domain.IdentityToken) (domain.Repository, error) {
	return naming.repository, nil
}

func presentedToken(t *testing.T) domain.IdentityToken {
	t.Helper()

	token, err := domain.ParseIdentityToken("presented-token")
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	return token
}

type unreadablePaths struct{ failure error }

func (unreadable unreadablePaths) KnownPathsOf(context.Context, domain.Repository) (domain.KnownPaths, error) {
	return domain.KnownPaths{}, unreadable.failure
}

func assertNothingStored(
	t *testing.T, maps domain.CoverageMaps, repository domain.Repository, suite domain.SuiteName,
) {
	t.Helper()

	if _, err := maps.SuiteCoverageOf(t.Context(), repository, suite); !errors.Is(err, domain.ErrNoCoverageMap) {
		t.Fatalf("a refused upload was stored: %v", err)
	}
}

type repositoryHolding []domain.RepositoryPath

func (held repositoryHolding) KnownPathsOf(context.Context, domain.Repository) (domain.KnownPaths, error) {
	return domain.NewKnownPaths(held), nil
}

type artifactReporting struct {
	reported []domain.ReportedFile
	reads    *int
}

func (artifact artifactReporting) ReportedFiles(domain.CoverageLimits) ([]domain.ReportedFile, error) {
	*artifact.reads++
	return artifact.reported, nil
}

func reportOf(path domain.RepositoryPath) artifactReporting {
	return artifactReporting{reported: []domain.ReportedFile{domain.NewReportedFile(path, true)}, reads: new(int)}
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
