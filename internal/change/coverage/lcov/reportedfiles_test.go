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

func TestReportedFilesInReadsAnExercisedRecord(t *testing.T) {
	artifact := "TN:\nSF:internal/change/domain/effect.go\nDA:7,3\nLF:1\nLH:1\nend_of_record\n"

	reported, err := lcov.ReportedFilesIn(strings.NewReader(artifact))

	if err != nil {
		t.Fatalf("parse one exercised record: %v", err)
	}
	want := []domain.ReportedFile{domain.NewReportedFile(repositoryPath(t, "internal/change/domain/effect.go"), true)}
	if !slices.Equal(reported, want) {
		t.Fatalf("reported %+v, want %+v", reported, want)
	}
}

func TestReportedFilesInRefusesAnAbsolutePathWhole(t *testing.T) {
	artifact := "SF:internal/change/domain/effect.go\nDA:7,3\nend_of_record\n" +
		"SF:/home/runner/work/effect.go\nDA:7,3\nend_of_record\n"

	reported, err := lcov.ReportedFilesIn(strings.NewReader(artifact))

	if !errors.Is(err, domain.ErrPathOutsideRepository) {
		t.Fatalf("want ErrPathOutsideRepository, got %v", err)
	}
	if reported != nil {
		t.Fatalf("want nothing reported alongside the refusal, got %+v", reported)
	}
}

func TestReportedFilesInRefusesATruncatedArtifactWhole(t *testing.T) {
	artifact := "SF:internal/change/domain/effect.go\nDA:7,3\nend_of_record\n" +
		"SF:internal/change/domain/gate.go\nDA:7,3\n"

	reported, err := lcov.ReportedFilesIn(strings.NewReader(artifact))

	if !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want ErrMalformedArtifact, got %v", err)
	}
	if reported != nil {
		t.Fatalf("want nothing reported alongside the refusal, got %+v", reported)
	}
}

func TestReportedFilesInRefusesAnArtifactWithNoRecord(t *testing.T) {
	_, err := lcov.ReportedFilesIn(strings.NewReader(""))

	if !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want ErrMalformedArtifact, got %v", err)
	}
}

func TestReportedFilesInRefusesARecordOutOfOrder(t *testing.T) {
	refused := map[string]string{
		"line data before any record":    "DA:7,3\nSF:internal/change/domain/effect.go\nend_of_record\n",
		"a record opened inside another": "SF:internal/change/domain/effect.go\nSF:internal/change/domain/gate.go\nend_of_record\n",
		"a record closed twice":          "SF:internal/change/domain/effect.go\nend_of_record\nend_of_record\n",
	}

	for form, artifact := range refused {
		t.Run(form, func(t *testing.T) {
			_, err := lcov.ReportedFilesIn(strings.NewReader(artifact))

			if !errors.Is(err, lcov.ErrMalformedArtifact) {
				t.Fatalf("want ErrMalformedArtifact, got %v", err)
			}
		})
	}
}

func repositoryPath(t *testing.T, path string) domain.RepositoryPath {
	t.Helper()

	parsed, err := domain.ParseRepositoryPath(path)
	if err != nil {
		t.Fatalf("parse repository path %q: %v", path, err)
	}
	return parsed
}
