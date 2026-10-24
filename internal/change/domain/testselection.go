// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

func (index coverageIndex) testsSelectedFor(evidence ChangeEvidence) ([]PackageTests, PlanTrigger, error) {
	if !index.holdsAnyTest() {
		return nil, PlanTriggerNoMap, nil
	}
	if evidence.holdsATestFileOfUnknownPackage() {
		return nil, PlanTriggerUnmappedFile, nil
	}
	tests, err := index.testsSelectedBy(evidence)
	if errors.Is(err, errUnindexedFile) {
		return nil, PlanTriggerUnmappedFile, nil
	}
	if err != nil {
		return nil, PlanTriggerPlanFailed, err
	}
	return tests, PlanTriggerNone, nil
}

func (index coverageIndex) testsSelectedBy(evidence ChangeEvidence) ([]PackageTests, error) {
	covering, err := index.testsCoveringChangesTo(evidence.filesBesideTests())
	if err != nil {
		return nil, err
	}
	placed := index.placementOf(evidence.tests)
	selected, err := covering.union(placed.indexed)
	if err != nil {
		return nil, err
	}
	selections := index.selectionsOf(selected)
	selections.addTests(placed.unindexed)
	selections.markWhole(evidence.packagesEditedBeyondTests)
	return index.packageTestsOf(selections)
}

func (index coverageIndex) testsCoveringChangesTo(files []ChangedFile) (testSet, error) {
	selected := testSetOf(index.revision, nil)
	for _, file := range files {
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

type placedTests struct {
	indexed   testSet
	unindexed []changedTest
}

func (index coverageIndex) placementOf(changed []changedTest) placedTests {
	var indexed []uint32
	var unindexed []changedTest
	for _, test := range changed {
		dense, isIndexed := index.positionOf(coveredTest{suite: CoverageSuite{importPath: test.pkg}, name: test.name})
		if isIndexed {
			indexed = append(indexed, uint32(dense))
			continue
		}
		unindexed = append(unindexed, test)
	}
	return placedTests{indexed: testSetOf(index.revision, indexed), unindexed: unindexed}
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

func (index coverageIndex) selectionsOf(selected testSet) packageSelections {
	selections := packageSelections{positionOf: map[CoverageSuite]int{}}
	for dense := range selected.members() {
		test := index.tests[dense]
		selection := selections.selectionFor(test.suite)
		selection.names = append(selection.names, test.name)
		selection.indexedCount++
	}
	return selections
}

func (index coverageIndex) packageTestsOf(selections packageSelections) ([]PackageTests, error) {
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
	if selection.isWhole || index.isEveryIndexedTestIn(selection) {
		return wholePackageOf(selection.suite.importPath), nil
	}
	return namedTestsOf(selection.suite.importPath, selection.names)
}

func (index coverageIndex) isEveryIndexedTestIn(selection packageSelection) bool {
	span, isIndexed := index.testsOfSuites[selection.suite]
	return isIndexed && selection.indexedCount == span.testCount()
}

type packageSelection struct {
	suite        CoverageSuite
	names        []TopLevelTestName
	indexedCount int
	isWhole      bool
}

type packageSelections struct {
	ordered    []packageSelection
	positionOf map[CoverageSuite]int
}

func (selections *packageSelections) addTests(changed []changedTest) {
	for _, test := range changed {
		selection := selections.selectionFor(CoverageSuite{importPath: test.pkg})
		selection.names = append(selection.names, test.name)
	}
}

func (selections *packageSelections) markWhole(packages []GoImportPath) {
	for _, pkg := range packages {
		selections.selectionFor(CoverageSuite{importPath: pkg}).isWhole = true
	}
}

func (selections *packageSelections) selectionFor(suite CoverageSuite) *packageSelection {
	position, isHeld := selections.positionOf[suite]
	if !isHeld {
		position = len(selections.ordered)
		selections.positionOf[suite] = position
		selections.ordered = append(selections.ordered, packageSelection{suite: suite})
	}
	return &selections.ordered[position]
}
