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

func TestOnlyATestThatFailedTwiceOutsideTheRecordedSelectionEscapedIt(t *testing.T) {
	selected, outside, alwaysRun := suiteOf(t, "example.com/app/rules"), suiteOf(t, "example.com/app/payments"), suiteOf(t, "example.com/app/domain")
	alwaysRunPath, err := domain.ParseGoImportPath(alwaysRun.String())
	if err != nil {
		t.Fatalf("parse import path: %v", err)
	}
	dispatched := domain.DispatchedPlan{
		AlwaysRun: []domain.GoImportPath{alwaysRunPath},
		Recorded:  domain.RecordedSets{IsRecorded: true, Selected: []domain.CoverageSuite{selected}},
	}
	escaped := domain.TestResult{Suite: outside, Name: "TestRefund", Outcome: domain.TestOutcomeFail, Rerun: domain.TestOutcomeFail}
	results := []domain.TestResult{
		escaped,
		{Suite: outside, Name: "TestCharge", Outcome: domain.TestOutcomeFail, Rerun: domain.TestOutcomePass},
		{Suite: outside, Name: "TestVoid", Outcome: domain.TestOutcomePass},
		{Suite: selected, Name: "TestReadFrom", Outcome: domain.TestOutcomeFail, Rerun: domain.TestOutcomeFail},
		{Suite: alwaysRun, Name: "TestEffect", Outcome: domain.TestOutcomeFail, Rerun: domain.TestOutcomeFail},
	}

	if got := dispatched.EscapesIn(results); len(got) != 1 || got[0].Test != escaped {
		t.Errorf("escapes are %+v, want only %+v", got, escaped)
	}
}

func suiteOf(t *testing.T, stated string) domain.CoverageSuite {
	t.Helper()

	suite, err := domain.ParseCoverageSuite(stated)
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	return suite
}

func TestAPlanThatRecordedNothingHasNoEscapes(t *testing.T) {
	failed := domain.TestResult{Suite: suiteOf(t, "example.com/app/payments"), Name: "TestRefund", Outcome: domain.TestOutcomeFail, Rerun: domain.TestOutcomeFail}

	if got := (domain.DispatchedPlan{}).EscapesIn([]domain.TestResult{failed}); len(got) != 0 {
		t.Errorf("a plan that recorded nothing gave escapes %+v, want none", got)
	}
}

func TestAnEscapeNamesItsModeThePullRequestTheTestAndTheChangedFiles(t *testing.T) {
	locator, err := domain.ParsePullRequestLocator("JoeNutt/open-ci-control-plane", 7)
	if err != nil {
		t.Fatalf("parse locator: %v", err)
	}
	changed, err := domain.ParseRepositoryPath("internal/change/rules/rules.go")
	if err != nil {
		t.Fatalf("parse path: %v", err)
	}
	dispatched := domain.DispatchedPlan{
		Locator: locator, ChangedFiles: []domain.RepositoryPath{changed},
		Recorded: domain.RecordedSets{IsRecorded: true, Selected: []domain.CoverageSuite{suiteOf(t, "example.com/app/rules")}},
	}
	failed := domain.TestResult{Suite: suiteOf(t, "example.com/app/payments"), Name: "TestRefund", Outcome: domain.TestOutcomeFail, Rerun: domain.TestOutcomeFail}

	escapes := dispatched.EscapesIn([]domain.TestResult{failed})

	if len(escapes) != 1 || escapes[0].Mode != domain.ShadowedSelection || escapes[0].PullRequest != locator ||
		escapes[0].Test != failed || !slices.Equal(escapes[0].ChangedFiles, []domain.RepositoryPath{changed}) {
		t.Errorf("escapes are %+v, want one against selection naming %s, %s and %s", escapes, locator, failed.Name, changed)
	}
}

func TestAMinimisationEscapeIsJudgedAgainstTheMinimisedSetOnlyWhenMinimisationIsInShadow(t *testing.T) {
	rules, payments := suiteOf(t, "example.com/app/rules"), suiteOf(t, "example.com/app/payments")
	failed := domain.TestResult{Suite: payments, Name: "TestRefund", Outcome: domain.TestOutcomeFail, Rerun: domain.TestOutcomeFail}
	for minimisation, want := range map[domain.TestMode][]domain.ShadowedMode{
		domain.TestModeShadow: {domain.ShadowedMinimisation},
		domain.TestModeOff:    nil,
	} {
		dispatched := domain.DispatchedPlan{
			Modes: domain.PlanModes{Selection: domain.TestModeShadow, Minimisation: minimisation},
			Recorded: domain.RecordedSets{IsRecorded: true,
				Selected: []domain.CoverageSuite{rules, payments}, Minimised: []domain.CoverageSuite{rules}},
		}

		var modes []domain.ShadowedMode
		for _, escape := range dispatched.EscapesIn([]domain.TestResult{failed}) {
			modes = append(modes, escape.Mode)
		}

		if !slices.Equal(modes, want) {
			t.Errorf("minimisation %s: escapes against %v, want %v", minimisation, modes, want)
		}
	}
}
