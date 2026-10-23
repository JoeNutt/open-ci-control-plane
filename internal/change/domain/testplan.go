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
	PlanTriggerPlanTooLarge PlanTrigger = "plan_too_large"
	PlanTriggerPlanFailed   PlanTrigger = "plan_failed"
)

const MaxPlanSuites = 512

type plannedSelection struct {
	diff      Diff
	selection coverageSelection
}

var fullMatrixTriggers = []struct {
	trigger PlanTrigger
	holds   func(plannedSelection) bool
}{
	{PlanTriggerNoMap, holdsNoMap},
	{PlanTriggerCIConfig, editsCIConfig},
	{PlanTriggerLockfile, editsLockfile},
	{PlanTriggerMigration, addsMigration},
	{PlanTriggerFileAdded, addsFile},
	{PlanTriggerUnmappedFile, changesUnmappedFile},
	{PlanTriggerPlanTooLarge, selectsTooManySuites},
}

func selectsTooManySuites(planned plannedSelection) bool {
	return len(planned.selection.suites) > MaxPlanSuites
}

func holdsNoMap(planned plannedSelection) bool { return !planned.selection.isAnyMapHeld }

func editsCIConfig(planned plannedSelection) bool { return len(planned.diff.Kinds.CIConfig) > 0 }

func editsLockfile(planned plannedSelection) bool { return len(planned.diff.Kinds.Lockfile) > 0 }

func addsFile(planned plannedSelection) bool { return slices.ContainsFunc(planned.diff.Files, isAdded) }

func changesUnmappedFile(planned plannedSelection) bool { return !planned.selection.isEveryFileMapped }

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
	Recorded       RecordedSets
}

type RecordedSets struct {
	IsRecorded bool
	Selected   []CoverageSuite
	Minimised  []CoverageSuite
}

type PlanSubject struct {
	Document   Document
	Binding    Binding
	DecisionID DecisionID
	Revision   BundleRevision
}

func PlanFor(subject PlanSubject, coverage RepositoryCoverage) (TestPlan, error) {
	head, err := ParseCommitSHA(subject.Document.PR.HeadSHA)
	if err != nil {
		return TestPlan{}, fmt.Errorf("plan tests: %w", err)
	}
	selection, err := coverage.selectionFor(subject.Document.Diff.Files)
	if err != nil {
		return TestPlan{}, fmt.Errorf("plan tests: %w", err)
	}
	trigger := triggerOf(plannedSelection{diff: subject.Document.Diff, selection: selection})
	return subject.shapedByMode(subject.planned(head, trigger, plannedSuitesOf(trigger, selection))), nil
}

var planShapedByMode = map[TestMode]func(PlanSubject, TestPlan) TestPlan{
	TestModeOff:     func(_ PlanSubject, plan TestPlan) TestPlan { return plan.asUnselected() },
	TestModeShadow:  PlanSubject.shadowed,
	TestModeEnforce: func(_ PlanSubject, plan TestPlan) TestPlan { return plan },
}

func (subject PlanSubject) shapedByMode(plan TestPlan) TestPlan {
	shaping, isKnown := planShapedByMode[subject.Binding.SelectionMode()]
	if !isKnown {
		return plan.asUnselected()
	}
	return shaping(subject, plan)
}

func (subject PlanSubject) shadowed(plan TestPlan) TestPlan {
	if plan.FullMatrix {
		return plan
	}
	return plan.asShadow(subject.recordedSetsOf(plan.Suites))
}

func FailedPlanFor(subject PlanSubject) (TestPlan, error) {
	head, err := ParseCommitSHA(subject.Document.PR.HeadSHA)
	if err != nil {
		return TestPlan{}, fmt.Errorf("plan the full matrix: %w", err)
	}
	return subject.planned(head, PlanTriggerPlanFailed, nil), nil
}

func (subject PlanSubject) planned(head CommitSHA, trigger PlanTrigger, suites []CoverageSuite) TestPlan {
	return TestPlan{
		DecisionID:     subject.DecisionID,
		Repository:     subject.Binding.Repository(),
		Head:           head,
		Workflow:       subject.Binding.Workflow(),
		Suites:         suites,
		AlwaysRun:      subject.Binding.AlwaysRun(),
		FullMatrix:     trigger != PlanTriggerNone,
		Trigger:        trigger,
		BundleRevision: subject.Revision,
	}
}

func plannedSuitesOf(trigger PlanTrigger, selection coverageSelection) []CoverageSuite {
	if trigger != PlanTriggerNone {
		return nil
	}
	return selection.suites
}

const MaxPlanCharacters = 60000

func (subject PlanSubject) recordedSetsOf(selected []CoverageSuite) RecordedSets {
	recorded := RecordedSets{IsRecorded: true, Selected: selected}
	if subject.Binding.MinimisationMode() == TestModeShadow {
		recorded.Minimised = minimisedOf(selected)
	}
	return recorded
}

func minimisedOf(selected []CoverageSuite) []CoverageSuite { return slices.Clone(selected) }

func (plan TestPlan) asShadow(recorded RecordedSets) TestPlan {
	plan = plan.asUnselected()
	plan.Recorded = recorded
	return plan
}

func (plan TestPlan) asUnselected() TestPlan {
	plan.Suites = nil
	plan.FullMatrix = true
	return plan
}

func (plan TestPlan) AsTooLarge() TestPlan {
	plan.Suites = nil
	plan.FullMatrix = true
	plan.Trigger = PlanTriggerPlanTooLarge
	return plan
}
