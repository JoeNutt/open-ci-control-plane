// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestTheNextPlanUsesTheCoverageMapCIJustUploaded(t *testing.T) {
	planner := plannerOverAnUpload(t)
	planner.Runs = provenRunsOf(t, locatorOf(t, uploadingRepository, 7))

	plan, err := planner.Execute(t.Context(), uploadedChangeOf(t))

	if err != nil {
		t.Fatalf("plan tests: %v", err)
	}
	if plan.FullMatrix || !slices.Equal(plan.Suites, []domain.CoverageSuite{parsedSuite(t)}) {
		t.Errorf("planned %v (full matrix %v, %q), want only the uploaded suite %s",
			plan.Suites, plan.FullMatrix, plan.Trigger, uploadedSuite)
	}
}

func TestAnEnforcedSelectionThatShadowRunsDoNotProveRunsTheFullMatrixAsUnproven(t *testing.T) {
	cases := map[string]heldRuns{
		"no shadow runs":                   nil,
		"another repository's shadow runs": provenRunsOf(t, gatedLocator(t)),
	}
	for name, runs := range cases {
		t.Run(name, func(t *testing.T) {
			planner := plannerOverAnUpload(t)
			planner.Runs = runs

			plan, err := planner.Execute(t.Context(), uploadedChangeOf(t))

			if err != nil || !plan.FullMatrix || plan.Trigger != domain.PlanTriggerShadowUnproven ||
				!reflect.DeepEqual(plan.Recorded.Selected, wholePackageOfSuite(t, parsedSuite(t))) {
				t.Errorf("planned %+v (%v), want the full matrix as %q recording the uploaded suite",
					plan, err, domain.PlanTriggerShadowUnproven)
			}
		})
	}
}

func TestAnEnforcedPlanWhoseShadowRunsCannotBeReadIsRefused(t *testing.T) {
	planner := plannerOverAnUpload(t)
	planner.Runs = unreadableRuns{}

	_, err := planner.Execute(t.Context(), uploadedChangeOf(t))

	if !errors.Is(err, domain.ErrHistoryFull) {
		t.Errorf("error is %v, want %v", err, domain.ErrHistoryFull)
	}
}

var plannedAt = time.Date(2026, time.November, 30, 12, 0, 0, 0, time.UTC)

func plannerOverAnUpload(t *testing.T) application.PlanTests {
	t.Helper()

	maps := newCoverageStore(domain.DeclaredCoverageLimits())
	upload := replaceCoverage(t, maps, repositoryHolding{parsedPath(t)}, tokensNaming{parsedRepository(t)})
	if _, err := upload.Execute(t.Context(), uploadOf(t, reportOf(t, parsedPath(t)))); err != nil {
		t.Fatalf("replace the suite's coverage: %v", err)
	}
	return application.PlanTests{
		Maps: maps, History: readableHistory{}, Escapes: heldEscapes{},
		Now: func() time.Time { return plannedAt },
	}
}

func uploadedChangeOf(t *testing.T) domain.PlanSubject {
	t.Helper()

	return domain.PlanSubject{
		Document: domain.Document{
			PR:   domain.PullRequest{Repository: uploadingRepository, HeadSHA: recordedHeadSHA},
			Diff: domain.Diff{Files: []domain.File{{Path: coveredFile, Action: domain.FileActionModified}}},
		},
		Binding:    uploadingRepositorysBinding(t),
		DecisionID: decisionIDOf(t, gatedDecisionID),
		Revision:   revisionForTest(t),
	}
}

func provenRunsOf(t *testing.T, locator domain.PullRequestLocator) heldRuns {
	t.Helper()

	inShadow := domain.DispatchedPlan{
		Locator: locator, Recorded: domain.RecordedSets{IsRecorded: true},
		Modes: domain.PlanModes{Selection: domain.TestModeShadow, Minimisation: domain.TestModeShadow},
	}
	var runs heldRuns
	for at := plannedAt.AddDate(0, -2, 0); at.Before(plannedAt); at = at.Add(time.Hour) {
		runs = append(runs, domain.ShadowedRun{At: at, Plan: inShadow, IsReported: true})
	}
	return runs
}

func TestAPlanIsRefusedWhenTheCoverageCannotBeRead(t *testing.T) {
	_, err := application.PlanTests{Maps: unreadableCoverage{}, History: readableHistory{}, Runs: heldRuns{}, Escapes: heldEscapes{}, Now: time.Now}.Execute(t.Context(), domain.PlanSubject{
		Document: domain.Document{PR: domain.PullRequest{Repository: uploadingRepository, HeadSHA: recordedHeadSHA}},
		Binding:  uploadingRepositorysBinding(t),
	})

	if !errors.Is(err, errCoverageUnreadable) {
		t.Errorf("error is %v, want %v", err, errCoverageUnreadable)
	}
}

var errCoverageUnreadable = errors.New("the coverage store cannot be read")

type unreadableCoverage struct {
	domain.CoverageMaps
}

func (unreadableCoverage) RepositoryCoverageOf(context.Context, domain.GovernedRepository) (domain.RepositoryCoverage, error) {
	return domain.RepositoryCoverage{}, errCoverageUnreadable
}

func uploadingRepositorysBinding(t *testing.T) domain.Binding {
	t.Helper()

	binding, err := domain.ParseBinding(domain.BindingStatement{
		Repository: uploadingRepository,
		Workflow:   "open-ci-tests.yml",
		AlwaysRun:  []string{uploadedSuite},
		Selection:  "enforce",
		Policies: []domain.BoundPolicyStatement{{
			Use:         "clean-architecture",
			ImportRoots: []domain.ImportRootStatement{{Prefix: "github.com/JoeNutt/open-ci-control-plane", Path: ""}},
			Paths:       map[string][]string{"domain": {"internal/change/domain/**"}},
		}},
	})
	if err != nil {
		t.Fatalf("parse binding: %v", err)
	}
	return binding
}
