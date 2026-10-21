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
	want := []domain.ReportedFile{{Path: "internal/change/domain/effect.go", IsExercised: true}}
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
