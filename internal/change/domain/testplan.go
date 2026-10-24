// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
)

type PlanTrigger string

const (
	PlanTriggerNone           PlanTrigger = "none"
	PlanTriggerUnmappedFile   PlanTrigger = "unmapped_file"
	PlanTriggerNoMap          PlanTrigger = "no_map"
	PlanTriggerFileAdded      PlanTrigger = "file_added"
	PlanTriggerLockfile       PlanTrigger = "lockfile"
	PlanTriggerMigration      PlanTrigger = "migration"
	PlanTriggerCIConfig       PlanTrigger = "ci_config"
	PlanTriggerPlanTooLarge   PlanTrigger = "plan_too_large"
	PlanTriggerPlanFailed     PlanTrigger = "plan_failed"
	PlanTriggerEscapeRecorded PlanTrigger = "escape_recorded"
	PlanTriggerShadowUnproven PlanTrigger = "shadow_unproven"
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
	Tests          []PackageTests
	AlwaysRun      []GoImportPath
	FullMatrix     bool
	Trigger        PlanTrigger
	BundleRevision BundleRevision
	Recorded       RecordedSets
	Modes          PlanModes
	ChangedFiles   []RepositoryPath
}

type PlanModes struct {
	Selection    TestMode
	Minimisation TestMode
}

type RecordedSets struct {
	IsRecorded bool
	Selected   []PackageTests
	Minimised  []PackageTests
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
	changed, err := changedPathsOf(subject.Document.Diff.Files)
	if err != nil {
		return TestPlan{}, fmt.Errorf("plan tests: %w", err)
	}
	plan := subject.planned(head, trigger, plannedSuitesOf(trigger, selection)).shapedByMode()
	plan.ChangedFiles = changed
	return plan, nil
}

func changedPathsOf(files []File) ([]RepositoryPath, error) {
	paths := make([]RepositoryPath, 0, len(files))
	for _, file := range files {
		path, err := ParseRepositoryPath(file.Path)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

var planShapedByMode = map[TestMode]func(TestPlan) TestPlan{
	TestModeOff:     TestPlan.asUnselected,
	TestModeShadow:  TestPlan.shadowed,
	TestModeEnforce: func(plan TestPlan) TestPlan { return plan },
}

func (plan TestPlan) shapedByMode() TestPlan {
	shaping, isKnown := planShapedByMode[plan.Modes.Selection]
	if !isKnown {
		return plan.asUnselected()
	}
	return shaping(plan)
}

func (plan TestPlan) shadowed() TestPlan {
	if plan.FullMatrix {
		return plan
	}
	return plan.asShadow(plan.recordedSets())
}

func (plan TestPlan) recordedSets() RecordedSets {
	recorded := RecordedSets{IsRecorded: true, Selected: plan.Tests}
	if plan.Modes.Minimisation == TestModeShadow {
		recorded.Minimised = minimisedOf(plan.Tests)
	}
	return recorded
}

func minimisedOf(selected []PackageTests) []PackageTests { return slices.Clone(selected) }

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
		Tests:          wholePackagesOf(suites),
		AlwaysRun:      subject.Binding.AlwaysRun(),
		FullMatrix:     trigger != PlanTriggerNone,
		Trigger:        trigger,
		BundleRevision: subject.Revision,
		Modes:          subject.Binding.modes(),
	}
}

func wholePackagesOf(suites []CoverageSuite) []PackageTests {
	var tests []PackageTests
	for _, suite := range suites {
		tests = append(tests, wholePackageOf(suite.importPath))
	}
	return tests
}

func plannedSuitesOf(trigger PlanTrigger, selection coverageSelection) []CoverageSuite {
	if trigger != PlanTriggerNone {
		return nil
	}
	return selection.suites
}

const MaxPlanCharacters = 60000

func (plan TestPlan) asShadow(recorded RecordedSets) TestPlan {
	plan = plan.asUnselected()
	plan.Recorded = recorded
	return plan
}

func (plan TestPlan) asUnselected() TestPlan {
	plan.Suites = nil
	plan.Tests = nil
	plan.FullMatrix = true
	return plan
}

func (plan TestPlan) AsWholePackages() TestPlan {
	var tests []PackageTests
	for _, entry := range plan.Tests {
		tests = append(tests, wholePackageOf(entry.pkg))
	}
	plan.Tests = tests
	var selected []PackageTests
	for _, entry := range plan.Recorded.Selected {
		selected = append(selected, wholePackageOf(entry.pkg))
	}
	plan.Recorded.Selected = selected
	var minimised []PackageTests
	for _, entry := range plan.Recorded.Minimised {
		minimised = append(minimised, wholePackageOf(entry.pkg))
	}
	plan.Recorded.Minimised = minimised
	return plan
}

func (plan TestPlan) AsTooLarge() TestPlan {
	plan = plan.asUnselected()
	plan.Recorded = RecordedSets{}
	plan.Trigger = PlanTriggerPlanTooLarge
	return plan
}
