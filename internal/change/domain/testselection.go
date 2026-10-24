// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

func (index coverageIndex) testsSelectedFor(evidence ChangeEvidence) ([]PackageTests, PlanTrigger, error) {
	if len(index.tests) == 0 {
		return nil, PlanTriggerNoMap, nil
	}
	selected, err := index.testsCoveringChangesTo(evidence.files)
	if errors.Is(err, errUnindexedFile) {
		return nil, PlanTriggerUnmappedFile, nil
	}
	if err != nil {
		return nil, PlanTriggerPlanFailed, err
	}
	tests, err := index.packageTestsOf(selected)
	if err != nil {
		return nil, PlanTriggerPlanFailed, err
	}
	return tests, PlanTriggerNone, nil
}

func (index coverageIndex) testsCoveringChangesTo(files []ChangedFile) (testSet, error) {
	selected := testSetOf(index.revision, nil)
	for _, file := range files {
		covering, err := index.testsCovering(file.base.path, file.lines.baseLineNumbers())
		if err != nil {
			return testSet{}, err
		}
		if selected, err = selected.union(covering); err != nil {
			return testSet{}, err
		}
	}
	return selected, nil
}

func (lines changedLines) baseLineNumbers() []uint32 {
	numbers := make([]uint32, 0, len(lines.base))
	for _, line := range lines.base {
		numbers = append(numbers, uint32(line))
	}
	return numbers
}

func (index coverageIndex) packageTestsOf(selected testSet) ([]PackageTests, error) {
	var suites []CoverageSuite
	namesBySuite := map[CoverageSuite][]TopLevelTestName{}
	for dense := range selected.members() {
		test := index.tests[dense]
		if _, isHeld := namesBySuite[test.suite]; !isHeld {
			suites = append(suites, test.suite)
		}
		namesBySuite[test.suite] = append(namesBySuite[test.suite], test.name)
	}
	tests := make([]PackageTests, 0, len(suites))
	for _, suite := range suites {
		inSuite, err := index.packageTestsIn(suite, namesBySuite[suite])
		if err != nil {
			return nil, err
		}
		tests = append(tests, inSuite)
	}
	return tests, nil
}

func (index coverageIndex) packageTestsIn(suite CoverageSuite, selected []TopLevelTestName) (PackageTests, error) {
	if len(selected) == index.testsOfSuites[suite].testCount() {
		return wholePackageOf(suite.importPath), nil
	}
	return namedTestsOf(suite.importPath, selected)
}
