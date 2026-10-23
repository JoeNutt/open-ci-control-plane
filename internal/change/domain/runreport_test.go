// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func reportedRepositoryAndHead(t *testing.T) (domain.GovernedRepository, domain.CommitSHA) {
	t.Helper()

	return governedRepositoryOf(t, "JoeNutt/open-ci-control-plane"), mustParseCommitSHA(t, planHeadSHA)
}

func TestARunReportCarriesItsRepositoryHeadAndResults(t *testing.T) {
	repository, head := reportedRepositoryAndHead(t)
	results := []domain.TestResult{{Name: "TestReadFrom", Outcome: domain.TestOutcomePass}}

	report, err := domain.ParseRunReport(repository, head, results)

	if err != nil {
		t.Fatalf("parse run report: %v", err)
	}
	if report.Repository != repository || report.Head != head || !slices.Equal(report.Results, results) {
		t.Errorf("parsed %+v, want %s at %s with %+v", report, repository, head, results)
	}
}

func TestARunReportIsRefusedAboveMaxReportedTests(t *testing.T) {
	repository, head := reportedRepositoryAndHead(t)

	for _, bounded := range []struct {
		count   int
		refusal error
	}{
		{domain.MaxReportedTests, nil},
		{domain.MaxReportedTests + 1, domain.ErrTooManyReportedTests},
	} {
		_, err := domain.ParseRunReport(repository, head, make([]domain.TestResult, bounded.count))

		if !errors.Is(err, bounded.refusal) {
			t.Errorf("a report of %d tests gave %v, want %v", bounded.count, err, bounded.refusal)
		}
	}
}
