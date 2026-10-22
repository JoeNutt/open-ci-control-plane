// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

type PlanTrigger string

const (
	PlanTriggerNone PlanTrigger = "none"
)

type TestPlan struct {
	DecisionID     DecisionID
	Repository     GovernedRepository
	Head           CommitSHA
	Workflow       WorkflowFile
	Suites         []CoverageSuite
	AlwaysRun      []GoImportPath
	FullMatrix     bool
	Trigger        PlanTrigger
	BundleRevision BundleRevision
}

type PlanBasis struct {
	Document   Document
	Coverage   RepositoryCoverage
	Binding    Binding
	DecisionID DecisionID
	Revision   BundleRevision
}

func PlanFor(basis PlanBasis) (TestPlan, error) {
	head, err := ParseCommitSHA(basis.Document.PR.HeadSHA)
	if err != nil {
		return TestPlan{}, fmt.Errorf("plan tests: %w", err)
	}
	suites, err := basis.Coverage.suitesCoveringAny(basis.Document.Diff.Files)
	if err != nil {
		return TestPlan{}, fmt.Errorf("plan tests: %w", err)
	}
	return TestPlan{
		DecisionID:     basis.DecisionID,
		Repository:     basis.Binding.Repository(),
		Head:           head,
		Workflow:       basis.Binding.Workflow(),
		Suites:         suites,
		AlwaysRun:      basis.Binding.AlwaysRun(),
		Trigger:        PlanTriggerNone,
		BundleRevision: basis.Revision,
	}, nil
}
