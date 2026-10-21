// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseSuiteCoverageCountsEveryReportedFile(t *testing.T) {
	exercised := mustParseRepositoryPath(t, "internal/change/domain/effect.go")
	unexercised := mustParseRepositoryPath(t, "internal/change/domain/gate.go")
	reported := []domain.ReportedFile{
		domain.NewReportedFile(exercised, true),
		domain.NewReportedFile(unexercised, false),
	}
	known := domain.NewKnownPaths([]domain.RepositoryPath{exercised, unexercised})

	coverage, err := domain.ParseSuiteCoverage(reported, known)

	if err != nil {
		t.Fatalf("parse coverage of held paths: %v", err)
	}
	if coverage.FileEntryCount() != len(reported) {
		t.Fatalf("file entry count is %d, want %d", coverage.FileEntryCount(), len(reported))
	}
}

func TestParseSuiteCoverageCoversOnlyExercisedFiles(t *testing.T) {
	exercised := mustParseRepositoryPath(t, "internal/change/domain/effect.go")
	unexercised := mustParseRepositoryPath(t, "internal/change/domain/gate.go")
	reported := []domain.ReportedFile{
		domain.NewReportedFile(exercised, true),
		domain.NewReportedFile(unexercised, false),
	}
	known := domain.NewKnownPaths([]domain.RepositoryPath{exercised, unexercised})

	coverage, err := domain.ParseSuiteCoverage(reported, known)

	if err != nil {
		t.Fatalf("parse coverage of held paths: %v", err)
	}
	if !coverage.Covers(exercised) {
		t.Errorf("an exercised file is not covered")
	}
	if coverage.Covers(unexercised) {
		t.Errorf("a file the suite never ran is covered")
	}
}

func TestParseSuiteCoverageRefusesAPathTheRepositoryDoesNotHold(t *testing.T) {
	held := mustParseRepositoryPath(t, "internal/change/domain/effect.go")
	absent := mustParseRepositoryPath(t, "internal/change/domain/missing.go")
	reported := []domain.ReportedFile{
		domain.NewReportedFile(held, true),
		domain.NewReportedFile(absent, true),
	}

	_, err := domain.ParseSuiteCoverage(reported, domain.NewKnownPaths([]domain.RepositoryPath{held}))

	if !errors.Is(err, domain.ErrPathOutsideRepository) {
		t.Fatalf("want ErrPathOutsideRepository, got %v", err)
	}
}
