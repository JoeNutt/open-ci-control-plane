// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/memory"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestTheNextPlanUsesTheCoverageMapCIJustUploaded(t *testing.T) {
	maps := memory.NewCoverageMaps(domain.DeclaredCoverageLimits())
	path := parsedPath(t)
	upload := replaceCoverage(t, maps, repositoryHolding{path}, tokensNaming{parsedRepository(t)})
	if _, err := upload.Execute(t.Context(), uploadOf(t, reportOf(path))); err != nil {
		t.Fatalf("replace the suite's coverage: %v", err)
	}

	plan, err := application.PlanTests{Maps: maps}.Execute(t.Context(), application.PlanRequest{
		Document: domain.Document{
			PR:   domain.PullRequest{Repository: uploadingRepository, HeadSHA: recordedHeadSHA},
			Diff: domain.Diff{Files: []domain.File{{Path: coveredFile, Action: domain.FileActionModified}}},
		},
		Binding:    uploadingRepositorysBinding(t),
		DecisionID: decisionIDOf(t, gatedDecisionID),
		Revision:   revisionForTest(t),
	})

	if err != nil {
		t.Fatalf("plan tests: %v", err)
	}
	if plan.FullMatrix || !slices.Equal(plan.Suites, []domain.CoverageSuite{parsedSuite(t)}) {
		t.Errorf("planned %v (full matrix %v, %q), want only the uploaded suite %s",
			plan.Suites, plan.FullMatrix, plan.Trigger, uploadedSuite)
	}
}

func uploadingRepositorysBinding(t *testing.T) domain.Binding {
	t.Helper()

	binding, err := domain.ParseBinding(domain.BindingStatement{
		Repository: uploadingRepository,
		Workflow:   "open-ci-tests.yml",
		AlwaysRun:  []string{uploadedSuite},
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
