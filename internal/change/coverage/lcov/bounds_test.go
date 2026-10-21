// SPDX-License-Identifier: GPL-3.0-or-later

package lcov_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/lcov"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const paddingLineBytes = 32 << 10

func TestReportedFilesInRefusesABodyLongerThanTheBoundWhateverItDeclares(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	artifact := lcov.Artifact{DeclaredBytes: 1, Body: paddedRecord(limits.MaxCompressedBytes + 1)}

	_, err := lcov.ReportedFilesIn(artifact, limits)

	if !errors.Is(err, domain.ErrCompressedArtifactTooLarge) {
		t.Fatalf("want ErrCompressedArtifactTooLarge, got %v", err)
	}
}

func paddedRecord(totalBytes int64) io.Reader {
	head := "SF:" + effectPath + "\n"
	tail := recordClose + "\n"
	parts := []io.Reader{strings.NewReader(head)}
	remaining := totalBytes - int64(len(head)+len(tail))
	fullLine := paddingLine(paddingLineBytes)
	for remaining > paddingLineBytes+minimalPaddingLineBytes {
		parts = append(parts, strings.NewReader(fullLine))
		remaining -= paddingLineBytes
	}
	if remaining > paddingLineBytes {
		parts = append(parts, strings.NewReader(paddingLine(remaining-minimalPaddingLineBytes)))
		remaining = minimalPaddingLineBytes
	}
	parts = append(parts, strings.NewReader(paddingLine(remaining)), strings.NewReader(tail))
	return io.MultiReader(parts...)
}

const minimalPaddingLineBytes = int64(len("LF:\n"))

func paddingLine(lineBytes int64) string {
	return "LF:" + strings.Repeat("0", int(lineBytes-minimalPaddingLineBytes)) + "\n"
}

type unreadBody struct{ t *testing.T }

func (body unreadBody) Read([]byte) (int, error) {
	body.t.Fatal("the body was read although its declared length is over the bound")
	return 0, nil
}

func TestReportedFilesInAcceptsADeclaredLengthAtTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	artifact := lcov.Artifact{
		DeclaredBytes: limits.MaxCompressedBytes,
		Body:          strings.NewReader(record(effectPath, lineHit)),
	}

	_, err := lcov.ReportedFilesIn(artifact, limits)

	if err != nil {
		t.Fatalf("want an artifact declared at the bound accepted, got %v", err)
	}
}

func TestReportedFilesInRefusesAnOversizedDeclaredLengthBeforeReading(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	artifact := lcov.Artifact{DeclaredBytes: limits.MaxCompressedBytes + 1, Body: unreadBody{t}}

	_, err := lcov.ReportedFilesIn(artifact, limits)

	if !errors.Is(err, domain.ErrCoverageBoundExceeded) {
		t.Fatalf("want ErrCoverageBoundExceeded, got %v", err)
	}
}
