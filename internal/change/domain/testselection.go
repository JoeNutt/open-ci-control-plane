// SPDX-License-Identifier: GPL-3.0-or-later

package domain

func (index coverageIndex) testsSelectedFor(evidence ChangeEvidence) ([]PackageTests, PlanTrigger, error) {
	selected := testSetOf(index.revision, nil)
	for _, file := range evidence.files {
		var lines []uint32
		for _, line := range file.lines.base {
			lines = append(lines, uint32(line))
		}
		covering, err := index.testsCovering(file.base.path, lines)
		if err != nil {
			return nil, "", err
		}
		selected, err = selected.union(covering)
		if err != nil {
			return nil, "", err
		}
	}
	namesBySuite := map[CoverageSuite][]TopLevelTestName{}
	var suites []CoverageSuite
	for dense := range selected.members() {
		test := index.tests[dense]
		if _, isHeld := namesBySuite[test.suite]; !isHeld {
			suites = append(suites, test.suite)
		}
		namesBySuite[test.suite] = append(namesBySuite[test.suite], test.name)
	}
	var tests []PackageTests
	for _, suite := range suites {
		named, err := namedTestsOf(suite.importPath, namesBySuite[suite])
		if err != nil {
			return nil, "", err
		}
		tests = append(tests, named)
	}
	return tests, PlanTriggerNone, nil
}
