// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

func (index coverageIndex) testsSelectedFor(evidence ChangeEvidence) ([]PackageTests, PlanTrigger, error) {
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
		span := index.testsOfSuites[suite]
		if len(namesBySuite[suite]) == span.end-span.first {
			tests = append(tests, wholePackageOf(suite.importPath))
			continue
		}
		named, err := namedTestsOf(suite.importPath, namesBySuite[suite])
		if err != nil {
			return nil, err
		}
		tests = append(tests, named)
	}
	return tests, nil
}
