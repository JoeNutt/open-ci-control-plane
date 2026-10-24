// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "testing"

func TestParsedSuiteCoverageFixesItsTestAndEdgeCounts(t *testing.T) {
	tests := []ReportedTest{
		testRunning(t, "TestReading", fileHitting(t, indexedEffectFile, 1, 2), fileHitting(t, indexedGateFile, 3)),
		testRunning(t, "TestBounds", fileHitting(t, indexedEffectFile, 2)),
	}
	paths := pathsRunBy(tests)

	coverage, err := ParseSuiteCoverage(NewReportedCoverage(paths, tests), NewKnownPaths(paths))

	if want := (indexSize{tests: 2, edges: 4}); err != nil || coverage.size != want {
		t.Fatalf("suite holds %+v (%v), want %+v", coverage.size, err, want)
	}
}
