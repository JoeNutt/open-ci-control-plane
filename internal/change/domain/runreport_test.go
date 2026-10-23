// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestARunReportCarriesItsRepositoryHeadAndResults(t *testing.T) {
	repository := governedRepositoryOf(t, "JoeNutt/open-ci-control-plane")
	head := mustParseCommitSHA(t, planHeadSHA)
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
	repository := governedRepositoryOf(t, "JoeNutt/open-ci-control-plane")
	head := mustParseCommitSHA(t, planHeadSHA)

	for _, bounded := range []struct {
		count     int
		isRefused bool
	}{
		{domain.MaxReportedTests, false},
		{domain.MaxReportedTests + 1, true},
	} {
		_, err := domain.ParseRunReport(repository, head, make([]domain.TestResult, bounded.count))

		if isRefused := err != nil; isRefused != bounded.isRefused {
			t.Errorf("a report of %d tests: refused %t (%v), want %t", bounded.count, isRefused, err, bounded.isRefused)
		}
	}
}
