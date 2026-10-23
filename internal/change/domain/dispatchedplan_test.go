// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestADispatchedPlanKeepsWhatItsPlanRecordedAndAlwaysRuns(t *testing.T) {
	suite, err := domain.ParseCoverageSuite(reportedTestSuite)
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	alwaysRun, err := domain.ParseGoImportPath("github.com/JoeNutt/open-ci-control-plane/internal/change/domain")
	if err != nil {
		t.Fatalf("parse import path: %v", err)
	}
	recorded := domain.RecordedSets{IsRecorded: true, Selected: []domain.CoverageSuite{suite}}
	changed, err := domain.ParseRepositoryPath("internal/change/rules/rules.go")
	if err != nil {
		t.Fatalf("parse path: %v", err)
	}

	dispatched := domain.DispatchedPlanOf(domain.PullRequestLocator{}, domain.TestPlan{
		Recorded: recorded, AlwaysRun: []domain.GoImportPath{alwaysRun}, ChangedFiles: []domain.RepositoryPath{changed},
	})

	if !dispatched.Recorded.IsRecorded || !slices.Equal(dispatched.Recorded.Selected, recorded.Selected) ||
		!slices.Equal(dispatched.AlwaysRun, []domain.GoImportPath{alwaysRun}) ||
		!slices.Equal(dispatched.ChangedFiles, []domain.RepositoryPath{changed}) {
		t.Errorf("the dispatched plan keeps %+v and always runs %v, want %+v and %v",
			dispatched.Recorded, dispatched.AlwaysRun, recorded, alwaysRun)
	}
}
