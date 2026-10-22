// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
)

type PlanTrigger string

const (
	PlanTriggerNone         PlanTrigger = "none"
	PlanTriggerUnmappedFile PlanTrigger = "unmapped_file"
	PlanTriggerNoMap        PlanTrigger = "no_map"
	PlanTriggerFileAdded    PlanTrigger = "file_added"
	PlanTriggerLockfile     PlanTrigger = "lockfile"
	PlanTriggerMigration    PlanTrigger = "migration"
	PlanTriggerCIConfig     PlanTrigger = "ci_config"
)

type plannedSelection struct {
	diff      Diff
	selection coverageSelection
}

var fullMatrixTriggers = []struct {
	trigger PlanTrigger
	holds   func(plannedSelection) bool
}{
	{PlanTriggerNoMap, func(planned plannedSelection) bool { return !planned.selection.isAnyMapHeld }},
	{PlanTriggerCIConfig, func(planned plannedSelection) bool { return len(planned.diff.Kinds.CIConfig) > 0 }},
	{PlanTriggerLockfile, func(planned plannedSelection) bool { return len(planned.diff.Kinds.Lockfile) > 0 }},
	{PlanTriggerMigration, addsMigration},
	{PlanTriggerFileAdded, func(planned plannedSelection) bool { return slices.ContainsFunc(planned.diff.Files, isAdded) }},
	{PlanTriggerUnmappedFile, func(planned plannedSelection) bool { return !planned.selection.isEveryFileMapped }},
}

func addsMigration(planned plannedSelection) bool {
	return slices.ContainsFunc(planned.diff.Files, func(file File) bool {
		return isAdded(file) && slices.Contains(planned.diff.Kinds.Migration, file.Path)
	})
}

func isAdded(file File) bool { return file.Action == FileActionAdded }

func triggerOf(planned plannedSelection) PlanTrigger {
	for _, rule := range fullMatrixTriggers {
		if rule.holds(planned) {
			return rule.trigger
		}
	}
	return PlanTriggerNone
}

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
	selection, err := basis.Coverage.selectionFor(basis.Document.Diff.Files)
	if err != nil {
		return TestPlan{}, fmt.Errorf("plan tests: %w", err)
	}
	trigger := triggerOf(plannedSelection{diff: basis.Document.Diff, selection: selection})
	return TestPlan{
		DecisionID:     basis.DecisionID,
		Repository:     basis.Binding.Repository(),
		Head:           head,
		Workflow:       basis.Binding.Workflow(),
		Suites:         selection.suites,
		AlwaysRun:      basis.Binding.AlwaysRun(),
		FullMatrix:     trigger != PlanTriggerNone,
		Trigger:        trigger,
		BundleRevision: basis.Revision,
	}, nil
}
