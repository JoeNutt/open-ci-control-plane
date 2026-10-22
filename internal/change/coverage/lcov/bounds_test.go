// SPDX-License-Identifier: GPL-3.0-or-later

package lcov_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/lcov"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	paddingTag              = "LF:"
	paddingLineBytes        = 32 << 10
	minimalPaddingLineBytes = int64(len(paddingTag + "\n"))
)

func TestReportedFilesAcceptsADeclaredLengthAtTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	artifact := lcov.Artifact{
		DeclaredBytes: limits.MaxCompressedBytes,
		Body:          strings.NewReader(record(effectPath, lineHit)),
	}

	_, err := artifact.ReportedFiles(limits)

	if err != nil {
		t.Fatalf("want an artifact declared at the bound accepted, got %v", err)
	}
}

func TestReportedFilesRefusesAnOversizedDeclaredLengthBeforeReading(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	artifact := lcov.Artifact{DeclaredBytes: limits.MaxCompressedBytes + 1, Body: unreadBody{t}}

	_, err := artifact.ReportedFiles(limits)

	assertBreachOf(t, domain.ErrCompressedArtifactTooLarge, err)
}

func TestReportedFilesAcceptsABodyAtTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	artifact := lcov.Artifact{
		DeclaredBytes: limits.MaxCompressedBytes,
		Body:          paddedRecord(limits.MaxCompressedBytes),
	}

	_, err := artifact.ReportedFiles(limits)

	if err != nil {
		t.Fatalf("want a body at the bound accepted, got %v", err)
	}
}

func TestReportedFilesRefusesABodyLongerThanTheBoundWhateverItDeclares(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	artifact := lcov.Artifact{DeclaredBytes: 1, Body: paddedRecord(limits.MaxCompressedBytes + 1)}

	_, err := artifact.ReportedFiles(limits)

	assertBreachOf(t, domain.ErrCompressedArtifactTooLarge, err)
}

func TestReportedFilesReadsAGzippedArtifact(t *testing.T) {
	compressed := gzipped(t, strings.NewReader(record(effectPath, lineHit)))
	artifact := lcov.Artifact{DeclaredBytes: int64(compressed.Len()), Body: compressed}

	reported, err := artifact.ReportedFiles(domain.DeclaredCoverageLimits())

	if err != nil {
		t.Fatalf("parse a gzipped record: %v", err)
	}
	want := []domain.ReportedFile{domain.NewReportedFile(repositoryPath(t, effectPath), true)}
	if !slices.Equal(reported, want) {
		t.Fatalf("reported %+v, want %+v", reported, want)
	}
}

func TestReportedFilesRefusesABrokenGzipStream(t *testing.T) {
	whole := gzipped(t, strings.NewReader(record(effectPath, lineHit))).Bytes()
	refused := map[string][]byte{
		"a corrupt header":   []byte("\x1f\x8bnot a gzip header"),
		"a truncated stream": whole[:len(whole)/2],
	}

	for form, body := range refused {
		t.Run(form, func(t *testing.T) {
			artifact := lcov.Artifact{DeclaredBytes: int64(len(body)), Body: bytes.NewReader(body)}

			_, err := artifact.ReportedFiles(domain.DeclaredCoverageLimits())

			if !errors.Is(err, lcov.ErrMalformedArtifact) {
				t.Fatalf("want ErrMalformedArtifact, got %v", err)
			}
		})
	}
}

func TestReportedFilesAcceptsAnArtifactExpandingToTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	compressed := gzipped(t, paddedRecord(limits.MaxDecompressedBytes))
	artifact := lcov.Artifact{DeclaredBytes: int64(compressed.Len()), Body: compressed}

	_, err := artifact.ReportedFiles(limits)

	if err != nil {
		t.Fatalf("want an artifact expanding to the bound accepted, got %v", err)
	}
}

func TestReportedFilesRefusesAnArtifactExpandingPastTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	compressed := gzipped(t, paddedRecord(limits.MaxDecompressedBytes+1))
	artifact := lcov.Artifact{DeclaredBytes: int64(compressed.Len()), Body: compressed}

	_, err := artifact.ReportedFiles(limits)

	assertBreachOf(t, domain.ErrDecompressedArtifactTooLarge, err)
}

func TestReportedFilesAcceptsFileEntriesAtTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()

	reported, err := artifactOfRecords(limits.MaxFileEntries).ReportedFiles(limits)

	if err != nil {
		t.Fatalf("want entries at the bound accepted, got %v", err)
	}
	if len(reported) != limits.MaxFileEntries {
		t.Fatalf("reported %d entries, want %d", len(reported), limits.MaxFileEntries)
	}
}

func TestReportedFilesRefusesMoreFileEntriesThanTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()

	_, err := artifactOfRecords(limits.MaxFileEntries + 1).ReportedFiles(limits)

	assertBreachOf(t, domain.ErrTooManyFileEntries, err)
}

func assertBreachOf(t *testing.T, bound, err error) {
	t.Helper()

	if !errors.Is(err, bound) {
		t.Fatalf("want %v, got %v", bound, err)
	}
	if errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("an artifact over a bound is not malformed: %v", err)
	}
}

type unreadBody struct{ t *testing.T }

func (body unreadBody) Read([]byte) (int, error) {
	body.t.Fatal("the body was read although its declared length is over the bound")
	return 0, nil
}

func artifactOfRecords(count int) lcov.Artifact {
	var records strings.Builder
	for index := range count {
		records.WriteString(record(fmt.Sprintf("internal/generated/file%d.go", index), lineHit))
	}
	return lcov.Artifact{DeclaredBytes: int64(records.Len()), Body: strings.NewReader(records.String())}
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

func paddingLine(lineBytes int64) string {
	return paddingTag + strings.Repeat("0", int(lineBytes-minimalPaddingLineBytes)) + "\n"
}
