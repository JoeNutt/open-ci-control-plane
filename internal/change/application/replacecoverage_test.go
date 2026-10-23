// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/lcov"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	uploadingRepository = "JoeNutt/open-ci-control-plane"
	strangerRepository  = "JoeNutt/open-ci-policy"
	uploadedSuite       = "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	coveredFile         = "internal/change/domain/effect.go"
	untouchedFile       = "internal/change/domain/verdict.go"
)

func TestReplaceCoverageStoresTheSuitesMap(t *testing.T) {
	maps := newCoverageStore(domain.DeclaredCoverageLimits())
	repository, suite, path := parsedRepository(t), parsedSuite(t), parsedPath(t)

	useCase := replaceCoverage(t, maps, repositoryHolding{path}, tokensNaming{repository})
	if _, err := useCase.Execute(t.Context(), uploadOf(t, reportOf(path))); err != nil {
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

func TestReplaceCoverageAnswersWithTheSuiteAndItsFileEntryCount(t *testing.T) {
	covered, untouched := parsedPath(t), parsedPathOf(t, untouchedFile)
	report := artifactReporting{reads: new(int), reported: []domain.ReportedFile{
		domain.NewReportedFile(covered, true), domain.NewReportedFile(untouched, false)}}
	useCase := replaceCoverage(t, newCoverageStore(domain.DeclaredCoverageLimits()), repositoryHolding{covered, untouched}, tokensNaming{parsedRepository(t)})

	replaced, err := useCase.Execute(t.Context(), uploadOf(t, report))

	if err != nil {
		t.Fatalf("replace the suite's coverage: %v", err)
	}
	if replaced.Suite != parsedSuite(t) || replaced.FileEntryCount != len(report.reported) {
		t.Fatalf("answered %s with %d file entries, want %s with %d",
			replaced.Suite, replaced.FileEntryCount, uploadedSuite, len(report.reported))
	}
}

func TestReplaceCoverageRefusesAPathTheRepositoryDoesNotHoldAndStoresNothing(t *testing.T) {
	maps := newCoverageStore(domain.DeclaredCoverageLimits())
	repository, suite, path := parsedRepository(t), parsedSuite(t), parsedPath(t)

	useCase := replaceCoverage(t, maps, repositoryHolding{}, tokensNaming{repository})
	_, err := useCase.Execute(t.Context(), uploadOf(t, reportOf(path)))

	if !errors.Is(err, domain.ErrPathOutsideRepository) {
		t.Fatalf("want ErrPathOutsideRepository, got %v", err)
	}
	assertNothingStored(t, maps, repository, suite)
}

func TestReplaceCoverageFailsWhenTheRepositorysPathsCannotBeRead(t *testing.T) {
	maps := newCoverageStore(domain.DeclaredCoverageLimits())
	repository, suite := parsedRepository(t), parsedSuite(t)
	unreadable := errors.New("the forge is unreachable")

	useCase := replaceCoverage(t, maps, unreadablePaths{unreadable}, tokensNaming{repository})
	_, err := useCase.Execute(t.Context(), uploadOf(t, reportOf(parsedPath(t))))

	if !errors.Is(err, unreadable) {
		t.Fatalf("want the read failure, got %v", err)
	}
	assertNothingStored(t, maps, repository, suite)
}

func TestReplaceCoverageRefusesATokenFromAnotherRepositoryAndStoresNothing(t *testing.T) {
	maps := newCoverageStore(domain.DeclaredCoverageLimits())
	repository, suite, path := parsedRepository(t), parsedSuite(t), parsedPath(t)

	useCase := replaceCoverage(t, maps, repositoryHolding{path}, tokensNaming{parsedStranger(t)})
	_, err := useCase.Execute(t.Context(), uploadOf(t, reportOf(path)))

	if !errors.Is(err, application.ErrForeignRepository) {
		t.Fatalf("want ErrForeignRepository, got %v", err)
	}
	assertNothingStored(t, maps, repository, suite)
}

func TestReplaceCoverageFailsWithTheVerifiersRefusalAndStoresNothing(t *testing.T) {
	maps := newCoverageStore(domain.DeclaredCoverageLimits())
	repository, suite, path := parsedRepository(t), parsedSuite(t), parsedPath(t)
	unverified := errors.New("token is absent, expired or not verifiable")

	useCase := replaceCoverage(t, maps, repositoryHolding{path}, unverifiedTokens{unverified})
	_, err := useCase.Execute(t.Context(), uploadOf(t, reportOf(path)))

	if !errors.Is(err, unverified) || errors.Is(err, application.ErrForeignRepository) {
		t.Fatalf("want only the verifier's refusal, got %v", err)
	}
	assertNothingStored(t, maps, repository, suite)
}

func TestReplaceCoverageNeverReadsTheArtifactOfARefusedUploader(t *testing.T) {
	path := parsedPath(t)
	artifact := reportOf(path)

	useCase := replaceCoverage(t, newCoverageStore(domain.DeclaredCoverageLimits()), repositoryHolding{path}, tokensNaming{parsedStranger(t)})
	_, err := useCase.Execute(t.Context(), uploadOf(t, artifact))

	if !errors.Is(err, application.ErrForeignRepository) {
		t.Fatalf("want ErrForeignRepository, got %v", err)
	}
	if *artifact.reads != 0 {
		t.Fatalf("read a refused uploader's artifact %d times", *artifact.reads)
	}
}

func TestReplaceCoverageRefusesAnUnreadableArtifactAndKeepsThePreviousMap(t *testing.T) {
	maps := newCoverageStore(domain.DeclaredCoverageLimits())
	repository, suite, path := parsedRepository(t), parsedSuite(t), parsedPath(t)
	useCase := replaceCoverage(t, maps, repositoryHolding{path}, tokensNaming{repository})
	if _, err := useCase.Execute(t.Context(), uploadOf(t, reportOf(path))); err != nil {
		t.Fatalf("accept the first map: %v", err)
	}

	_, err := useCase.Execute(t.Context(), uploadOf(t, unreadableArtifact{}))

	if !errors.Is(err, domain.ErrTooManyFileEntries) {
		t.Fatalf("want ErrTooManyFileEntries, got %v", err)
	}
	served, err := maps.SuiteCoverageOf(t.Context(), repository, suite)
	if err != nil || !served.Covers(path) {
		t.Fatalf("the previous map stopped serving: covers %s is false, err %v", path, err)
	}
}

func TestReplaceCoverageKeepsServingTheMapATruncatedLCOVArtifactWouldReplace(t *testing.T) {
	maps := newCoverageStore(domain.DeclaredCoverageLimits())
	path := parsedPath(t)
	useCase := replaceCoverage(t, maps, repositoryHolding{path}, tokensNaming{parsedRepository(t)})
	whole := "TN:TestEffect\nSF:" + coveredFile + "\nDA:7,3\nend_of_record\n"
	if _, err := useCase.Execute(t.Context(), uploadOf(t, lcovArtifact(whole))); err != nil {
		t.Fatalf("accept the whole artifact: %v", err)
	}

	_, err := useCase.Execute(t.Context(), uploadOf(t, lcovArtifact("TN:TestEffect\nSF:"+coveredFile+"\nDA:7,0\n")))

	if !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want ErrMalformedArtifact, got %v", err)
	}
	served, err := maps.SuiteCoverageOf(t.Context(), parsedRepository(t), parsedSuite(t))
	if err != nil || !served.Covers(path) {
		t.Fatalf("the previous map stopped serving: covers %s is false, err %v", path, err)
	}
}

func lcovArtifact(body string) lcov.Artifact {
	return lcov.Artifact{DeclaredBytes: int64(len(body)), Body: strings.NewReader(body)}
}

func TestReplaceCoverageRefusesAnUploadOverTheRateWithoutReadingIt(t *testing.T) {
	maps := newCoverageStore(domain.DeclaredCoverageLimits())
	path := parsedPath(t)
	rate := rateFor(t, domain.DeclaredCoverageLimits(), func() time.Time { return windowOpens })
	admitAll(t, rate, parsedRepository(t), domain.DeclaredCoverageLimits().MaxUploadsPerMinute)
	artifact := reportOf(path)
	useCase := application.ReplaceCoverage{
		Maps: maps, Paths: repositoryHolding{path}, Callers: tokensNaming{parsedRepository(t)}, Rate: rate}

	_, err := useCase.Execute(t.Context(), uploadOf(t, artifact))

	if !errors.Is(err, application.ErrUploadRateExceeded) {
		t.Fatalf("want ErrUploadRateExceeded, got %v", err)
	}
	if *artifact.reads != 0 {
		t.Fatalf("read an artifact over the rate %d times", *artifact.reads)
	}
	assertNothingStored(t, maps, parsedRepository(t), parsedSuite(t))
}

func TestReplaceCoverageNeverSpendsTheAllowanceOnARefusedUploader(t *testing.T) {
	path := parsedPath(t)
	maps := newCoverageStore(domain.DeclaredCoverageLimits())
	rate := freshRate(t)
	strangers := application.ReplaceCoverage{
		Maps: maps, Paths: repositoryHolding{path}, Callers: tokensNaming{parsedStranger(t)}, Rate: rate}
	for range domain.DeclaredCoverageLimits().MaxUploadsPerMinute {
		if _, err := strangers.Execute(t.Context(), uploadOf(t, reportOf(path))); !errors.Is(err, application.ErrForeignRepository) {
			t.Fatalf("want ErrForeignRepository, got %v", err)
		}
	}
	governed := application.ReplaceCoverage{
		Maps: maps, Paths: repositoryHolding{path}, Callers: tokensNaming{parsedRepository(t)}, Rate: rate}

	if _, err := governed.Execute(t.Context(), uploadOf(t, reportOf(path))); err != nil {
		t.Fatalf("refused uploaders spent the repository's allowance: %v", err)
	}
}

type unreadableArtifact struct{}

func (unreadableArtifact) ReportedFiles(domain.CoverageLimits) ([]domain.ReportedFile, error) {
	return nil, domain.ErrTooManyFileEntries
}

type unverifiedTokens struct{ refusal error }

func (unverified unverifiedTokens) RepositoryOf(context.Context, domain.IdentityToken) (domain.GovernedRepository, error) {
	return domain.GovernedRepository{}, unverified.refusal
}

type tokensNaming struct{ repository domain.GovernedRepository }

func (naming tokensNaming) RepositoryOf(context.Context, domain.IdentityToken) (domain.GovernedRepository, error) {
	return naming.repository, nil
}

func uploadOf(t *testing.T, report domain.CoverageReport) application.CoverageUpload {
	t.Helper()

	return application.CoverageUpload{
		Token: presentedToken(t), Repository: parsedRepository(t), Suite: parsedSuite(t), Report: report}
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

func (unreadable unreadablePaths) KnownPathsOf(context.Context, domain.GovernedRepository) (domain.KnownPaths, error) {
	return domain.KnownPaths{}, unreadable.failure
}

func replaceCoverage(
	t *testing.T, maps domain.CoverageMaps, paths domain.RepositoryPaths, callers domain.CallerRepositories,
) application.ReplaceCoverage {
	t.Helper()

	return application.ReplaceCoverage{Maps: maps, Paths: paths, Callers: callers, Rate: freshRate(t)}
}

func freshRate(t *testing.T) *application.UploadRate {
	t.Helper()

	return rateFor(t, domain.DeclaredCoverageLimits(), func() time.Time { return windowOpens })
}

func assertNothingStored(
	t *testing.T, maps domain.CoverageMaps, repository domain.GovernedRepository, suite domain.CoverageSuite,
) {
	t.Helper()

	if _, err := maps.SuiteCoverageOf(t.Context(), repository, suite); !errors.Is(err, domain.ErrNoCoverageMap) {
		t.Fatalf("a refused upload was stored: %v", err)
	}
}

type repositoryHolding []domain.RepositoryPath

func (held repositoryHolding) KnownPathsOf(context.Context, domain.GovernedRepository) (domain.KnownPaths, error) {
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

func parsedRepository(t *testing.T) domain.GovernedRepository {
	t.Helper()

	repository, err := domain.ParseGovernedRepository(uploadingRepository)
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}
	return repository
}

func parsedStranger(t *testing.T) domain.GovernedRepository {
	t.Helper()

	stranger, err := domain.ParseGovernedRepository(strangerRepository)
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}
	return stranger
}

func parsedSuite(t *testing.T) domain.CoverageSuite {
	t.Helper()

	suite, err := domain.ParseCoverageSuite(uploadedSuite)
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	return suite
}

func parsedPath(t *testing.T) domain.RepositoryPath {
	t.Helper()

	return parsedPathOf(t, coveredFile)
}

func parsedPathOf(t *testing.T, file string) domain.RepositoryPath {
	t.Helper()

	path, err := domain.ParseRepositoryPath(file)
	if err != nil {
		t.Fatalf("parse path: %v", err)
	}
	return path
}
