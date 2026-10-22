// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	planHeadSHA   = "9f1cabc0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6"
	ordersSuite   = "github.com/JoeNutt/open-ci-control-plane/internal/orders"
	paymentsSuite = "github.com/JoeNutt/open-ci-control-plane/internal/payments"
	ordersFile    = "internal/orders/order.go"
	paymentsFile  = "internal/payments/charge.go"
)

func TestAChangeToFilesTwoSuitesCoverPlansThoseSuitesAndTheAlwaysRunSet(t *testing.T) {
	plan := planOf(t, plannedChange{
		files:    []domain.File{modified(ordersFile), modified(paymentsFile)},
		coverage: coverageByFiles(t, map[string]string{ordersSuite: ordersFile, paymentsSuite: paymentsFile}),
	})

	if got := suiteNamesOf(plan.Suites); !slices.Equal(got, []string{ordersSuite, paymentsSuite}) {
		t.Errorf("suites are %v, want %v", got, []string{ordersSuite, paymentsSuite})
	}
	if !slices.Equal(plan.AlwaysRun, planBinding(t).AlwaysRun()) {
		t.Errorf("always-run set is %v, want the binding's %v", plan.AlwaysRun, planBinding(t).AlwaysRun())
	}
	if plan.FullMatrix || plan.Trigger != domain.PlanTriggerNone {
		t.Errorf("full matrix %v with trigger %q, want a selected plan with %q", plan.FullMatrix, plan.Trigger, domain.PlanTriggerNone)
	}
}

func TestEachReasonSelectionCannotVouchForPlansTheFullMatrixNamingIt(t *testing.T) {
	mapped := coverageByFiles(t, map[string]string{ordersSuite: ordersFile, paymentsSuite: paymentsFile})
	cases := map[string]struct {
		change plannedChange
		want   domain.PlanTrigger
	}{
		"a changed file in no map": {plannedChange{
			files: []domain.File{modified(ordersFile), modified("README.md")}, coverage: mapped,
		}, domain.PlanTriggerUnmappedFile},
		"no map at all": {plannedChange{
			files: []domain.File{modified(ordersFile)},
		}, domain.PlanTriggerNoMap},
		"an added file": {plannedChange{
			files: []domain.File{added(ordersFile)}, coverage: mapped,
		}, domain.PlanTriggerFileAdded},
		"a lockfile": {plannedChange{
			files: []domain.File{modified(ordersFile)}, coverage: mapped, kinds: domain.Kinds{Lockfile: domain.Paths{"go.sum"}},
		}, domain.PlanTriggerLockfile},
		"an added migration": {plannedChange{
			files: []domain.File{added(ordersFile)}, coverage: mapped, kinds: domain.Kinds{Migration: domain.Paths{ordersFile}},
		}, domain.PlanTriggerMigration},
		"CI configuration": {plannedChange{
			files: []domain.File{modified(ordersFile)}, coverage: mapped, kinds: domain.Kinds{CIConfig: domain.Paths{".github/workflows/ci.yml"}},
		}, domain.PlanTriggerCIConfig},
		"precedence: no map over every other reason": {plannedChange{
			files: []domain.File{added("README.md")}, kinds: domain.Kinds{CIConfig: domain.Paths{"README.md"}},
		}, domain.PlanTriggerNoMap},
		"precedence: CI configuration over a lockfile": {plannedChange{
			files: []domain.File{modified(ordersFile)}, coverage: mapped,
			kinds: domain.Kinds{CIConfig: domain.Paths{".github/workflows/ci.yml"}, Lockfile: domain.Paths{"go.sum"}},
		}, domain.PlanTriggerCIConfig},
	}

	for form, planned := range cases {
		t.Run(form, func(t *testing.T) {
			plan := planOf(t, planned.change)

			if !plan.FullMatrix || plan.Trigger != planned.want {
				t.Errorf("full matrix %v with trigger %q, want the full matrix with %q", plan.FullMatrix, plan.Trigger, planned.want)
			}
			if !slices.Equal(plan.AlwaysRun, planBinding(t).AlwaysRun()) {
				t.Errorf("always-run set is %v, want the binding's %v", plan.AlwaysRun, planBinding(t).AlwaysRun())
			}
		})
	}
}

func TestAPlanCarriesItsDecisionRepositoryHeadRevisionAndTheBindingsWorkflow(t *testing.T) {
	plan := planOf(t, plannedChange{
		files: []domain.File{modified(".github/workflows/open-ci-tests.yml"), added(".openci.yaml")},
	})

	binding := planBinding(t)
	if plan.DecisionID != decisionIDForPlan(t) || plan.Repository != binding.Repository() {
		t.Errorf("decision %q in %q, want %q in %q", plan.DecisionID, plan.Repository, decisionIDForPlan(t), binding.Repository())
	}
	if plan.Head.String() != planHeadSHA || plan.BundleRevision != revisionForTest(t) {
		t.Errorf("head %q at revision %q, want %q at %q", plan.Head, plan.BundleRevision, planHeadSHA, revisionForTest(t))
	}
	if plan.Workflow != binding.Workflow() || !slices.Equal(plan.AlwaysRun, binding.AlwaysRun()) {
		t.Errorf("workflow %q running %v, want the binding's %q running %v",
			plan.Workflow, plan.AlwaysRun, binding.Workflow(), binding.AlwaysRun())
	}
}

func TestAPlanAtTheSuiteLimitIsSelectedAndOneSuiteOverIsTheFullMatrix(t *testing.T) {
	for _, bound := range []struct {
		suites     int
		fullMatrix bool
		trigger    domain.PlanTrigger
	}{
		{domain.MaxPlanSuites, false, domain.PlanTriggerNone},
		{domain.MaxPlanSuites + 1, true, domain.PlanTriggerPlanTooLarge},
	} {
		plan := planOf(t, changeCoveredBySuites(t, bound.suites))

		if plan.FullMatrix != bound.fullMatrix || plan.Trigger != bound.trigger {
			t.Errorf("%d suites: full matrix %v with %q, want %v with %q",
				bound.suites, plan.FullMatrix, plan.Trigger, bound.fullMatrix, bound.trigger)
		}
		if plan.FullMatrix && len(plan.Suites) != 0 {
			t.Errorf("%d suites: a full-matrix plan names %d suites, want none", bound.suites, len(plan.Suites))
		}
	}
}

func changeCoveredBySuites(t *testing.T, count int) plannedChange {
	t.Helper()

	fileBySuite := make(map[string]string, count)
	files := make([]domain.File, 0, count)
	for index := range count {
		file := fmt.Sprintf("internal/suite%d/file.go", index)
		fileBySuite[fmt.Sprintf("github.com/JoeNutt/open-ci-control-plane/internal/suite%d", index)] = file
		files = append(files, modified(file))
	}
	return plannedChange{files: files, coverage: coverageByFiles(t, fileBySuite)}
}

type plannedChange struct {
	files    []domain.File
	coverage domain.RepositoryCoverage
	kinds    domain.Kinds
}

func planOf(t *testing.T, change plannedChange) domain.TestPlan {
	t.Helper()

	plan, err := domain.PlanFor(domain.PlanSubject{
		Document: domain.Document{
			PR:   domain.PullRequest{Repository: thisRepository, HeadSHA: planHeadSHA},
			Diff: domain.Diff{Files: change.files, Kinds: change.kinds},
		},
		Binding:    planBinding(t),
		DecisionID: decisionIDForPlan(t),
		Revision:   revisionForTest(t),
	}, change.coverage)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	return plan
}

func modified(path string) domain.File {
	return domain.File{Path: path, Action: domain.FileActionModified}
}

func added(path string) domain.File {
	return domain.File{Path: path, Action: domain.FileActionAdded}
}

func coverageByFiles(t *testing.T, fileBySuite map[string]string) domain.RepositoryCoverage {
	t.Helper()

	var held domain.RepositoryCoverage
	for suite, file := range fileBySuite {
		path, err := domain.ParseRepositoryPath(file)
		if err != nil {
			t.Fatalf("parse path: %v", err)
		}
		coverage, err := domain.ParseSuiteCoverage(
			[]domain.ReportedFile{domain.NewReportedFile(path, true)}, domain.NewKnownPaths([]domain.RepositoryPath{path}))
		if err != nil {
			t.Fatalf("parse suite coverage: %v", err)
		}
		held = mustReplace(t, held, mustParseSuite(t, suite), coverage)
	}
	return held
}

func planBinding(t *testing.T) domain.Binding {
	t.Helper()

	binding, err := domain.ParseBinding(thisRepositorysBinding())
	if err != nil {
		t.Fatalf("parse binding: %v", err)
	}
	return binding
}

func decisionIDForPlan(t *testing.T) domain.DecisionID {
	t.Helper()

	id, err := domain.ParseDecisionID("dec_0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("parse decision ID: %v", err)
	}
	return id
}

func suiteNamesOf(suites []domain.CoverageSuite) []string {
	names := make([]string, 0, len(suites))
	for _, suite := range suites {
		names = append(names, suite.String())
	}
	return names
}
