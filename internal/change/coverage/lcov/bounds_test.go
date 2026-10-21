// SPDX-License-Identifier: GPL-3.0-or-later

package lcov_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/lcov"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type unreadBody struct{ t *testing.T }

func (body unreadBody) Read([]byte) (int, error) {
	body.t.Fatal("the body was read although its declared length is over the bound")
	return 0, nil
}

func TestReportedFilesInRefusesAnOversizedDeclaredLengthBeforeReading(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	artifact := lcov.Artifact{DeclaredBytes: limits.MaxCompressedBytes + 1, Body: unreadBody{t}}

	_, err := lcov.ReportedFilesIn(artifact, limits)

	if !errors.Is(err, domain.ErrCoverageBoundExceeded) {
		t.Fatalf("want ErrCoverageBoundExceeded, got %v", err)
	}
}
