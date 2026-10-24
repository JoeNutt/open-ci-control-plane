// SPDX-License-Identifier: GPL-3.0-or-later

package lcov_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/lcov"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	effectPath    = "internal/change/domain/effect.go"
	gatePath      = "internal/change/domain/gate.go"
	runnerPath    = "/home/runner/work/effect.go"
	lineHitNumber = 7
	recordClose   = "end_of_record"
	effectTest    = "TestEffect"
	gateTest      = "TestGate"
)

var (
	lineHit    = fmt.Sprintf("DA:%d,3", lineHitNumber)
	lineMissed = fmt.Sprintf("DA:%d,0", lineHitNumber)
)

func TestReportedTestsReadAnExercisedRecord(t *testing.T) {
	artifact := testSection(effectTest, lines("SF:"+effectPath, lineHit, "LF:1", "LH:1", recordClose))

	reported, err := reportedTestsIn(artifact)

	if err != nil {
		t.Fatalf("parse one exercised record: %v", err)
	}
	want := []domain.ReportedTest{reportedTest(t, effectTest, testedFile(t, effectPath, lineHitNumber))}
	if !reflect.DeepEqual(reported, want) {
		t.Fatalf("reported %+v, want %+v", reported, want)
	}
}

func TestReportedTestsReportEachRecordOnItsOwnCounts(t *testing.T) {
	artifact := testSection(effectTest, record(effectPath, lineHit), record(gatePath, lineMissed, "DA:8,0"))

	reported, err := reportedTestsIn(artifact)

	if err != nil {
		t.Fatalf("parse two records: %v", err)
	}
	want := []domain.ReportedTest{
		reportedTest(t, effectTest, testedFile(t, effectPath, lineHitNumber), testedFile(t, gatePath)),
	}
	if !reflect.DeepEqual(reported, want) {
		t.Fatalf("reported %+v, want %+v", reported, want)
	}
}

func TestReportedTestsReportAFileOnceForEachTestThatRanIt(t *testing.T) {
	artifact := testSection(effectTest, record(effectPath, lineMissed), record(gatePath, lineMissed)) +
		testSection(gateTest, record(effectPath, lineHit), record(gatePath, lineMissed))

	reported, err := reportedTestsIn(artifact)

	if err != nil {
		t.Fatalf("parse two test sections: %v", err)
	}
	want := []domain.ReportedTest{
		reportedTest(t, effectTest, testedFile(t, effectPath), testedFile(t, gatePath)),
		reportedTest(t, gateTest, testedFile(t, effectPath, lineHitNumber), testedFile(t, gatePath)),
	}
	if !reflect.DeepEqual(reported, want) {
		t.Fatalf("reported %+v, want %+v", reported, want)
	}
}

func TestReportedTestsReadTheArtifactCIUploads(t *testing.T) {
	artifact := testSection(effectTest, record(effectPath, "DA:7,1", "DA:8,4"), record(gatePath, "DA:3,1")) +
		testSection(gateTest, record(gatePath, "DA:3,2", "DA:9,1"))

	reported, err := reportedTestsIn(artifact)

	if err != nil {
		t.Fatalf("parse the artifact CI uploads: %v", err)
	}
	want := []domain.ReportedTest{
		reportedTest(t, effectTest, testedFile(t, effectPath, 7, 8), testedFile(t, gatePath, 3)),
		reportedTest(t, gateTest, testedFile(t, gatePath, 3, 9)),
	}
	if !reflect.DeepEqual(reported, want) {
		t.Fatalf("reported %+v, want %+v", reported, want)
	}
}

func TestReportedTestsRefuseAnAbsolutePathWhole(t *testing.T) {
	err := refusalOf(t, testSection(effectTest, record(effectPath, lineHit), record(runnerPath, lineHit)))

	if !errors.Is(err, domain.ErrPathOutsideRepository) {
		t.Fatalf("want ErrPathOutsideRepository, got %v", err)
	}
	if errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("a readable artifact with an absolute path is not malformed: %v", err)
	}
}

func TestReportedTestsRefuseATruncatedArtifactWhole(t *testing.T) {
	err := refusalOf(t, testSection(effectTest, record(effectPath, lineHit), lines("SF:"+gatePath, lineHit)))

	if !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want ErrMalformedArtifact, got %v", err)
	}
}

func TestReportedTestsRefuseAnArtifactWithNoRecord(t *testing.T) {
	err := refusalOf(t, "")

	if !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want ErrMalformedArtifact, got %v", err)
	}
}

func TestReportedTestsRefuseAnArtifactWithNoTestSectionWhole(t *testing.T) {
	err := refusalOf(t, record(effectPath, lineHit))

	if !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want ErrMalformedArtifact, got %v", err)
	}
}

func TestReportedTestsRefuseARecordOutsideATestSectionWhole(t *testing.T) {
	err := refusalOf(t, record(gatePath, lineHit)+testSection(effectTest, record(effectPath, lineHit)))

	if !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want ErrMalformedArtifact, got %v", err)
	}
}

func TestReportedTestsRefuseATestSectionWithNoNameWhole(t *testing.T) {
	err := refusalOf(t, testSection(effectTest, record(gatePath, lineHit))+testSection("", record(effectPath, lineHit)))

	if !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want ErrMalformedArtifact, got %v", err)
	}
}

func TestReportedTestsRefuseASectionNamingNoTopLevelTestWhole(t *testing.T) {
	for _, name := range []string{"BenchmarkEffect", "test", "Testé", "Test-Effect", "Test Effect", "TestEffect/sub", "TestEffect\x00"} {
		t.Run(name, func(t *testing.T) {
			err := refusalOf(t, testSection(name, record(effectPath, lineHit)))

			if !errors.Is(err, lcov.ErrMalformedArtifact) {
				t.Fatalf("want ErrMalformedArtifact, got %v", err)
			}
		})
	}
}

func TestReportedTestsRefuseATestNamedByTwoSectionsWhole(t *testing.T) {
	err := refusalOf(t, testSection(effectTest, record(effectPath, lineHit))+
		testSection(gateTest, record(gatePath, lineHit))+testSection(effectTest, record(gatePath, lineHit)))

	if !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want ErrMalformedArtifact, got %v", err)
	}
}

func TestReportedTestsRefuseARecordOutOfOrder(t *testing.T) {
	refused := map[string]string{
		"line data before any record":    testSection(effectTest, lines(lineHit), record(effectPath)),
		"a record opened inside another": testSection(effectTest, lines("SF:"+effectPath), record(gatePath)),
		"a record closed twice":          testSection(effectTest, record(effectPath), lines(recordClose)),
	}

	for form, artifact := range refused {
		t.Run(form, func(t *testing.T) {
			err := refusalOf(t, artifact)

			if !errors.Is(err, lcov.ErrMalformedArtifact) || !errors.Is(err, domain.ErrMalformedCoverageReport) {
				t.Fatalf("want ErrMalformedArtifact, got %v", err)
			}
		})
	}
}

func TestReportedTestsRefuseAMalformedLine(t *testing.T) {
	refused := map[string]string{
		"an unknown tag":                  "XX:1",
		"line data with no count":         "DA:7",
		"line data with a word for count": "DA:7,many",
		"line data with a negative count": "DA:7,-1",
		"hit line data at line zero":      "DA:0,1",
		"hit line data with a word line":  "DA:seven,1",
		"hit line data past 32 bits":      "DA:4294967296,1",
		"missed line data at line zero":   "DA:0,0",
		"missed line data with a word":    "DA:seven,0",
		"missed line data past 32 bits":   "DA:4294967296,0",
	}

	for form, line := range refused {
		t.Run(form, func(t *testing.T) {
			err := refusalOf(t, testSection(effectTest, record(effectPath, line)))

			if !errors.Is(err, lcov.ErrMalformedArtifact) {
				t.Fatalf("want ErrMalformedArtifact, got %v", err)
			}
		})
	}
}

func refusalOf(t *testing.T, artifact string) error {
	t.Helper()

	reported, err := reportedTestsIn(artifact)
	if reported != nil {
		t.Fatalf("want nothing reported alongside a refusal, got %+v", reported)
	}
	return err
}

func reportedTestsIn(artifact string) ([]domain.ReportedTest, error) {
	var report domain.CoverageReport = artifactOf(artifact)
	return report.ReportedTests(domain.DeclaredCoverageLimits())
}

func artifactOf(text string) lcov.Artifact {
	return lcov.Artifact{DeclaredBytes: int64(len(text)), Body: strings.NewReader(text)}
}

func testSection(name string, records ...string) string {
	return lines("TN:"+name) + strings.Join(records, "")
}

func record(path string, body ...string) string {
	return lines("SF:"+path) + lines(body...) + lines(recordClose)
}

func lines(each ...string) string {
	var joined strings.Builder
	for _, line := range each {
		joined.WriteString(line + "\n")
	}
	return joined.String()
}

func repositoryPath(t *testing.T, path string) domain.RepositoryPath {
	t.Helper()

	parsed, err := domain.ParseRepositoryPath(path)
	if err != nil {
		t.Fatalf("parse repository path %q: %v", path, err)
	}
	return parsed
}

func TestReportedTestsKeepEachSectionsFilesAndTheLinesItHit(t *testing.T) {
	artifact := testSection(effectTest, record(effectPath, "DA:7,1", "DA:8,0"), record(gatePath, "DA:3,0")) +
		testSection(gateTest, record(gatePath, "DA:9,1", "DA:3,2"))

	reported, err := reportedTestsIn(artifact)

	if err != nil {
		t.Fatalf("parse two test sections: %v", err)
	}
	want := []domain.ReportedTest{
		reportedTest(t, effectTest, testedFile(t, effectPath, 7), testedFile(t, gatePath)),
		reportedTest(t, gateTest, testedFile(t, gatePath, 3, 9)),
	}
	if !reflect.DeepEqual(reported, want) {
		t.Fatalf("reported %+v, want %+v", reported, want)
	}
}

func reportedTest(t *testing.T, name string, files ...domain.TestedFile) domain.ReportedTest {
	t.Helper()

	return domain.NewReportedTest(testName(t, name), files)
}

func testedFile(t *testing.T, path string, hitLines ...uint32) domain.TestedFile {
	t.Helper()

	return domain.NewTestedFile(repositoryPath(t, path), hitLines)
}

func testName(t *testing.T, stated string) domain.TopLevelTestName {
	t.Helper()

	name, err := domain.ParseTopLevelTestName(stated)
	if err != nil {
		t.Fatalf("parse test name %q: %v", stated, err)
	}
	return name
}
