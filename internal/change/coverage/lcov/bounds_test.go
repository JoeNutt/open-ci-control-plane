// SPDX-License-Identifier: GPL-3.0-or-later

package lcov_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"slices"
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
	if errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("an artifact over a bound is not malformed: %v", err)
	}
}

func TestReportedFilesInAcceptsABodyAtTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	artifact := lcov.Artifact{
		DeclaredBytes: limits.MaxCompressedBytes,
		Body:          paddedRecord(limits.MaxCompressedBytes),
	}

	_, err := lcov.ReportedFilesIn(artifact, limits)

	if err != nil {
		t.Fatalf("want a body at the bound accepted, got %v", err)
	}
}

func TestReportedFilesInReadsAGzippedArtifact(t *testing.T) {
	compressed := gzipped(t, strings.NewReader(record(effectPath, lineHit)))
	artifact := lcov.Artifact{DeclaredBytes: int64(compressed.Len()), Body: compressed}

	reported, err := lcov.ReportedFilesIn(artifact, domain.DeclaredCoverageLimits())

	if err != nil {
		t.Fatalf("parse a gzipped record: %v", err)
	}
	want := []domain.ReportedFile{domain.NewReportedFile(repositoryPath(t, effectPath), true)}
	if !slices.Equal(reported, want) {
		t.Fatalf("reported %+v, want %+v", reported, want)
	}
}

func TestReportedFilesInRefusesAnArtifactExpandingPastTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	compressed := gzipped(t, paddedRecord(limits.MaxDecompressedBytes+1))
	artifact := lcov.Artifact{DeclaredBytes: int64(compressed.Len()), Body: compressed}

	_, err := lcov.ReportedFilesIn(artifact, limits)

	if !errors.Is(err, domain.ErrDecompressedArtifactTooLarge) {
		t.Fatalf("want ErrDecompressedArtifactTooLarge, got %v", err)
	}
	if errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("an artifact over a bound is not malformed: %v", err)
	}
}

func gzipped(t *testing.T, plain io.Reader) *bytes.Buffer {
	t.Helper()

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := io.Copy(writer, plain); err != nil {
		t.Fatalf("compress: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("finish compressing: %v", err)
	}
	return &compressed
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
