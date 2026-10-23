// SPDX-License-Identifier: GPL-3.0-or-later

package lcov_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/lcov"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	effectPath  = "internal/change/domain/effect.go"
	gatePath    = "internal/change/domain/gate.go"
	runnerPath  = "/home/runner/work/effect.go"
	lineHit     = "DA:7,3"
	lineMissed  = "DA:7,0"
	recordClose = "end_of_record"
	effectTest  = "TestEffect"
)

func TestReportedFilesReadsAnExercisedRecord(t *testing.T) {
	artifact := testSection(effectTest, lines("SF:"+effectPath, lineHit, "LF:1", "LH:1", recordClose))

	reported, err := reportedFilesIn(artifact)

	if err != nil {
		t.Fatalf("parse one exercised record: %v", err)
	}
	want := []domain.ReportedFile{domain.NewReportedFile(repositoryPath(t, effectPath), true)}
	if !slices.Equal(reported, want) {
		t.Fatalf("reported %+v, want %+v", reported, want)
	}
}

func TestReportedFilesReportsEachRecordOnItsOwnCounts(t *testing.T) {
	artifact := testSection(effectTest, record(effectPath, lineHit), record(gatePath, lineMissed, "DA:8,0"))

	reported, err := reportedFilesIn(artifact)

	if err != nil {
		t.Fatalf("parse two records: %v", err)
	}
	want := []domain.ReportedFile{
		domain.NewReportedFile(repositoryPath(t, effectPath), true),
		domain.NewReportedFile(repositoryPath(t, gatePath), false),
	}
	if !slices.Equal(reported, want) {
		t.Fatalf("reported %+v, want %+v", reported, want)
	}
}

func TestReportedFilesReportsTheUnionOfItsTestsOncePerFile(t *testing.T) {
	artifact := testSection(effectTest, record(effectPath, lineMissed), record(gatePath, lineMissed)) +
		testSection("TestGate", record(effectPath, lineHit), record(gatePath, lineMissed))

	reported, err := reportedFilesIn(artifact)

	if err != nil {
		t.Fatalf("parse two test sections: %v", err)
	}
	want := []domain.ReportedFile{
		domain.NewReportedFile(repositoryPath(t, effectPath), true),
		domain.NewReportedFile(repositoryPath(t, gatePath), false),
	}
	if !slices.Equal(reported, want) {
		t.Fatalf("reported %+v, want %+v", reported, want)
	}
}

func TestReportedFilesRefusesAnAbsolutePathWhole(t *testing.T) {
	err := refusalOf(t, testSection(effectTest, record(effectPath, lineHit), record(runnerPath, lineHit)))

	if !errors.Is(err, domain.ErrPathOutsideRepository) {
		t.Fatalf("want ErrPathOutsideRepository, got %v", err)
	}
	if errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("a readable artifact with an absolute path is not malformed: %v", err)
	}
}

func TestReportedFilesRefusesATruncatedArtifactWhole(t *testing.T) {
	err := refusalOf(t, testSection(effectTest, record(effectPath, lineHit), lines("SF:"+gatePath, lineHit)))

	if !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want ErrMalformedArtifact, got %v", err)
	}
}

func TestReportedFilesRefusesAnArtifactWithNoRecord(t *testing.T) {
	err := refusalOf(t, "")

	if !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want ErrMalformedArtifact, got %v", err)
	}
}

func TestReportedFilesRefusesAnArtifactWithNoTestSectionWhole(t *testing.T) {
	err := refusalOf(t, record(effectPath, lineHit))

	if !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want ErrMalformedArtifact, got %v", err)
	}
}

func TestReportedFilesRefusesARecordOutsideATestSectionWhole(t *testing.T) {
	err := refusalOf(t, record(gatePath, lineHit)+testSection(effectTest, record(effectPath, lineHit)))

	if !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want ErrMalformedArtifact, got %v", err)
	}
}

func TestReportedFilesRefusesATestSectionWithNoNameWhole(t *testing.T) {
	err := refusalOf(t, testSection(effectTest, record(gatePath, lineHit))+testSection("", record(effectPath, lineHit)))

	if !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want ErrMalformedArtifact, got %v", err)
	}
}

func TestReportedFilesRefusesARecordOutOfOrder(t *testing.T) {
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

func TestReportedFilesRefusesAMalformedLine(t *testing.T) {
	refused := map[string]string{
		"an unknown tag":                  "XX:1",
		"line data with no count":         "DA:7",
		"line data with a word for count": "DA:7,many",
		"line data with a negative count": "DA:7,-1",
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

	reported, err := reportedFilesIn(artifact)
	if reported != nil {
		t.Fatalf("want nothing reported alongside a refusal, got %+v", reported)
	}
	return err
}

func reportedFilesIn(artifact string) ([]domain.ReportedFile, error) {
	var report domain.CoverageReport = lcov.Artifact{
		DeclaredBytes: int64(len(artifact)), Body: strings.NewReader(artifact)}
	return report.ReportedFiles(domain.DeclaredCoverageLimits())
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
