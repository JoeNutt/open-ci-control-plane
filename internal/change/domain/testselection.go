// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"slices"
)

func (index coverageIndex) testsSelectedFor(evidence ChangeEvidence) ([]PackageTests, PlanTrigger, error) {
	if !index.holdsAnyTest() {
		return nil, PlanTriggerNoMap, nil
	}
	selected, err := index.testsCoveringChangesTo(evidence.files, evidence.testFiles)
	if errors.Is(err, errUnindexedFile) {
		return nil, PlanTriggerUnmappedFile, nil
	}
	if err != nil {
		return nil, PlanTriggerPlanFailed, err
	}
	tests, err := index.packageTestsOf(selected, evidence.tests)
	if err != nil {
		return nil, PlanTriggerPlanFailed, err
	}
	return tests, PlanTriggerNone, nil
}

func (index coverageIndex) testsCoveringChangesTo(files []ChangedFile, testFiles []RepositoryPath) (testSet, error) {
	selected := testSetOf(index.revision, nil)
	for _, file := range files {
		if slices.Contains(testFiles, file.path) {
			continue
		}
		covering, err := index.testsTouchedBy(file)
		if err != nil {
			return testSet{}, err
		}
		if selected, err = selected.union(covering); err != nil {
			return testSet{}, err
		}
	}
	return selected, nil
}

func (index coverageIndex) testsTouchedBy(file ChangedFile) (testSet, error) {
	if !file.lines.isKnown || file.action.leavesItsBasePath() {
		return index.testsReaching(file.base.path)
	}
	return index.testsCovering(file.base.path, file.lines.baseLineNumbers())
}

func (lines changedLines) baseLineNumbers() []uint32 {
	numbers := make([]uint32, 0, len(lines.base))
	for _, line := range lines.base {
		numbers = append(numbers, uint32(line))
	}
	return numbers
}

func (index coverageIndex) packageTestsOf(selected testSet, changed []changedTest) ([]PackageTests, error) {
	selections := packageSelections{positionOf: map[CoverageSuite]int{}}
	for dense := range selected.members() {
		test := index.tests[dense]
		selections.adding(test.suite, test.name).indexedCount++
	}
	for _, test := range changed {
		selections.adding(CoverageSuite{importPath: test.pkg}, test.name)
	}
	tests := make([]PackageTests, 0, len(selections.ordered))
	for _, selection := range selections.ordered {
		inPackage, err := index.packageTestsIn(selection)
		if err != nil {
			return nil, err
		}
		tests = append(tests, inPackage)
	}
	return tests, nil
}

func (index coverageIndex) packageTestsIn(selection packageSelection) (PackageTests, error) {
	span, isIndexed := index.testsOfSuites[selection.suite]
	if isIndexed && selection.indexedCount == span.testCount() {
		return wholePackageOf(selection.suite.importPath), nil
	}
	return namedTestsOf(selection.suite.importPath, selection.names)
}

type packageSelection struct {
	suite        CoverageSuite
	names        []TopLevelTestName
	indexedCount int
}

type packageSelections struct {
	ordered    []packageSelection
	positionOf map[CoverageSuite]int
}

func (selections *packageSelections) adding(suite CoverageSuite, name TopLevelTestName) *packageSelection {
	position, isHeld := selections.positionOf[suite]
	if !isHeld {
		position = len(selections.ordered)
		selections.positionOf[suite] = position
		selections.ordered = append(selections.ordered, packageSelection{suite: suite})
	}
	selections.ordered[position].names = append(selections.ordered[position].names, name)
	return &selections.ordered[position]
}
