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
