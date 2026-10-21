// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

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
