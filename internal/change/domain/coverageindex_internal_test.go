// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"testing"
)

const (
	indexedDomainSuite = "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	indexedLCOVSuite   = "github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/lcov"
	indexedEffectFile  = "internal/change/domain/effect.go"
	indexedGateFile    = "internal/change/domain/gate.go"
)

func TestCoverageIndexNumbersEveryTestDenselyBySuiteThenName(t *testing.T) {
	coverage := twoSuiteCoverage(t)
	want := []coveredTest{
		{suite: suiteNamed(t, indexedLCOVSuite), name: testNamed(t, "TestBounds")},
		{suite: suiteNamed(t, indexedLCOVSuite), name: testNamed(t, "TestReading")},
		{suite: suiteNamed(t, indexedDomainSuite), name: testNamed(t, "TestGate")},
	}

	for dense, wanted := range want {
		test, err := coverage.index.testAt(dense)
		if err != nil || test != wanted {
			t.Errorf("test %d is %+v (%v), want %+v", dense, test, err, wanted)
		}
	}
}

func TestCoverageIndexRefusesAPositionOutsideItsTests(t *testing.T) {
	coverage := twoSuiteCoverage(t)

	for _, dense := range []int{-1, len(coverage.index.tests)} {
		if _, err := coverage.index.testAt(dense); !errors.Is(err, errNoTestAtIndex) {
			t.Errorf("position %d: want errNoTestAtIndex, got %v", dense, err)
		}
	}
}

func TestCoverageIndexCountsTheTestsEachSuiteHolds(t *testing.T) {
	coverage := twoSuiteCoverage(t)

	for suite, want := range map[string]int{indexedLCOVSuite: 2, indexedDomainSuite: 1} {
		if count, err := coverage.index.testCountOf(suiteNamed(t, suite)); err != nil || count != want {
			t.Errorf("%s holds %d tests (%v), want %d", suite, count, err, want)
		}
	}
}

func TestCoverageIndexRefusesToCountASuiteItHoldsNoCoverageFor(t *testing.T) {
	unindexed := suiteNamed(t, "github.com/JoeNutt/open-ci-control-plane/internal/change/httpapi")

	count, err := twoSuiteCoverage(t).index.testCountOf(unindexed)

	if !errors.Is(err, errUnindexedSuite) {
		t.Fatalf("counted %d tests (%v), want errUnindexedSuite", count, err)
	}
}

func twoSuiteCoverage(t *testing.T) RepositoryCoverage {
	t.Helper()

	return coverageHolding(t, map[string][]ReportedTest{
		indexedDomainSuite: {testRunning(t, "TestGate", fileHitting(t, indexedGateFile, 1))},
		indexedLCOVSuite: {
			testRunning(t, "TestReading", fileHitting(t, indexedEffectFile, 1)),
			testRunning(t, "TestBounds", fileHitting(t, indexedEffectFile, 2)),
		},
	})
}

func coverageHolding(t *testing.T, testsBySuite map[string][]ReportedTest) RepositoryCoverage {
	t.Helper()

	var coverage RepositoryCoverage
	for suite, tests := range testsBySuite {
		known := NewKnownPaths(pathsRunBy(tests))
		suiteCoverage, err := ParseSuiteCoverage(tests, known)
		if err != nil {
			t.Fatalf("parse coverage of %s: %v", suite, err)
		}
		coverage, err = coverage.Replacing(suiteNamed(t, suite), suiteCoverage, DeclaredCoverageLimits())
		if err != nil {
			t.Fatalf("replace %s: %v", suite, err)
		}
	}
	return coverage
}

func pathsRunBy(tests []ReportedTest) []RepositoryPath {
	var paths []RepositoryPath
	for _, test := range tests {
		for _, file := range test.files {
			paths = append(paths, file.path)
		}
	}
	return paths
}

func testRunning(t *testing.T, name string, files ...TestedFile) ReportedTest {
	t.Helper()

	return NewReportedTest(testNamed(t, name), files)
}

func fileHitting(t *testing.T, path string, lines ...uint32) TestedFile {
	t.Helper()

	parsed, err := ParseRepositoryPath(path)
	if err != nil {
		t.Fatalf("parse path %q: %v", path, err)
	}
	return NewTestedFile(parsed, lines)
}

func testNamed(t *testing.T, name string) TopLevelTestName {
	t.Helper()

	parsed, err := ParseTopLevelTestName(name)
	if err != nil {
		t.Fatalf("parse test name %q: %v", name, err)
	}
	return parsed
}

func suiteNamed(t *testing.T, suite string) CoverageSuite {
	t.Helper()

	parsed, err := ParseCoverageSuite(suite)
	if err != nil {
		t.Fatalf("parse suite %q: %v", suite, err)
	}
	return parsed
}
