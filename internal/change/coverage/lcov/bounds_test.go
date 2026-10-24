// SPDX-License-Identifier: GPL-3.0-or-later

package lcov_test

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"reflect"
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

func TestReportedTestsAcceptsADeclaredLengthAtTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	artifact := lcov.Artifact{
		DeclaredBytes: limits.MaxCompressedBytes,
		Body:          strings.NewReader(testSection(effectTest, record(effectPath, lineHit))),
	}

	_, err := artifact.ReportedCoverage(limits)

	if err != nil {
		t.Fatalf("want an artifact declared at the bound accepted, got %v", err)
	}
}

func TestReportedTestsRefusesAnOversizedDeclaredLengthBeforeReading(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	artifact := lcov.Artifact{DeclaredBytes: limits.MaxCompressedBytes + 1, Body: unreadBody{t}}

	_, err := artifact.ReportedCoverage(limits)

	assertBreachOf(t, domain.ErrCompressedArtifactTooLarge, err)
}

func TestReportedTestsAcceptsABodyAtTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	artifact := lcov.Artifact{
		DeclaredBytes: limits.MaxCompressedBytes,
		Body:          paddedRecord(limits.MaxCompressedBytes),
	}

	_, err := artifact.ReportedCoverage(limits)

	if err != nil {
		t.Fatalf("want a body at the bound accepted, got %v", err)
	}
}

func TestReportedTestsRefusesABodyLongerThanTheBoundWhateverItDeclares(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	artifact := lcov.Artifact{DeclaredBytes: 1, Body: paddedRecord(limits.MaxCompressedBytes + 1)}

	_, err := artifact.ReportedCoverage(limits)

	assertBreachOf(t, domain.ErrCompressedArtifactTooLarge, err)
}

func TestReportedTestsReadsAGzippedArtifact(t *testing.T) {
	compressed := gzipped(t, strings.NewReader(testSection(effectTest, record(effectPath, lineHit))))
	artifact := lcov.Artifact{DeclaredBytes: int64(compressed.Len()), Body: compressed}

	reported, err := artifact.ReportedCoverage(domain.DeclaredCoverageLimits())

	if err != nil {
		t.Fatalf("parse a gzipped record: %v", err)
	}
	want := reportedCoverage(t, []string{effectPath}, reportedTest(t, effectTest, testedFile(t, effectPath, lineHitNumber)))
	if !reflect.DeepEqual(reported, want) {
		t.Fatalf("reported %+v, want %+v", reported, want)
	}
}

func TestReportedTestsRefusesABrokenGzipStream(t *testing.T) {
	whole := gzipped(t, strings.NewReader(record(effectPath, lineHit))).Bytes()
	refused := map[string][]byte{
		"a corrupt header":   []byte("\x1f\x8bnot a gzip header"),
		"a truncated stream": whole[:len(whole)/2],
	}

	for form, body := range refused {
		t.Run(form, func(t *testing.T) {
			artifact := lcov.Artifact{DeclaredBytes: int64(len(body)), Body: bytes.NewReader(body)}

			_, err := artifact.ReportedCoverage(domain.DeclaredCoverageLimits())

			if !errors.Is(err, lcov.ErrMalformedArtifact) {
				t.Fatalf("want ErrMalformedArtifact, got %v", err)
			}
		})
	}
}

func TestReportedTestsAcceptsAnArtifactExpandingToTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	compressed := gzipped(t, paddedRecord(limits.MaxDecompressedBytes))
	artifact := lcov.Artifact{DeclaredBytes: int64(compressed.Len()), Body: compressed}

	_, err := artifact.ReportedCoverage(limits)

	if err != nil {
		t.Fatalf("want an artifact expanding to the bound accepted, got %v", err)
	}
}

func TestReportedTestsRefusesAnArtifactExpandingPastTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	compressed := gzipped(t, paddedRecord(limits.MaxDecompressedBytes+1))
	artifact := lcov.Artifact{DeclaredBytes: int64(compressed.Len()), Body: compressed}

	_, err := artifact.ReportedCoverage(limits)

	assertBreachOf(t, domain.ErrDecompressedArtifactTooLarge, err)
}

func TestReportedTestsAcceptsFileEntriesAtTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()

	_, err := artifactOfRecords(limits.MaxFileEntries).ReportedCoverage(limits)

	if err != nil {
		t.Fatalf("want entries at the bound accepted, got %v", err)
	}
}

func TestReportedTestsCountsAFileReportedByAnotherTestAsNoNewEntry(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	atTheBound := artifactOfRecords(limits.MaxFileEntries)
	again := testSection(generatedTest(0), record(generatedPath(0), lineHit))
	artifact := lcov.Artifact{Body: io.MultiReader(atTheBound.Body, strings.NewReader(again))}

	_, err := artifact.ReportedCoverage(limits)

	if err != nil {
		t.Fatalf("want a file reported again accepted at the bound, got %v", err)
	}
}

func TestReportedTestsRefusesMoreFileEntriesThanTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()

	_, err := artifactOfRecords(limits.MaxFileEntries + 1).ReportedCoverage(limits)

	assertBreachOf(t, domain.ErrTooManyFileEntries, err)
}

func TestReportedTestsAcceptsTestSectionsAtTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()

	_, err := artifactOfTestSections(limits.MaxTestSections).ReportedCoverage(limits)

	if err != nil {
		t.Fatalf("want sections at the bound accepted, got %v", err)
	}
}

func TestReportedTestsRefusesMoreTestSectionsThanTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()

	_, err := artifactOfTestSections(limits.MaxTestSections + 1).ReportedCoverage(limits)

	assertBreachOf(t, domain.ErrTooManyTestSections, err)
}

func TestReportedTestsAcceptsLineRecordsAtTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()

	_, err := artifactOfLineRecords(limits.MaxLineRecords).ReportedCoverage(limits)

	if err != nil {
		t.Fatalf("want DA records at the bound accepted, got %v", err)
	}
}

func TestReportedTestsRefusesMoreLineRecordsThanTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()

	_, err := artifactOfLineRecords(limits.MaxLineRecords + 1).ReportedCoverage(limits)

	assertBreachOf(t, domain.ErrTooManyLineRecords, err)
}

func TestReportedTestsHoldTestFilePairsUpToTheLineRecordBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()

	_, err := artifactOfTestFilePairs(t, limits.MaxLineRecords).ReportedCoverage(limits)

	if err != nil {
		t.Fatalf("want one hit record per test and file accepted up to the bound, got %v", err)
	}
}

func TestReportedTestsRefuseOneTestFilePairPastTheLineRecordBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()

	_, err := artifactOfTestFilePairs(t, limits.MaxLineRecords+1).ReportedCoverage(limits)

	assertBreachOf(t, domain.ErrTooManyLineRecords, err)
}

func TestReportedTestsRetainNoRecordThatHitNoLine(t *testing.T) {
	unexercised := slices.Repeat([]string{record(effectPath, lineMissed)}, 3)
	artifact := testSection(effectTest, unexercised...) + testSection(gateTest, unexercised...)

	reported, err := reportedTestsIn(artifact)

	want := reportedCoverage(t, []string{effectPath}, reportedTest(t, effectTest), reportedTest(t, gateTest))
	if err != nil || !reflect.DeepEqual(reported, want) {
		t.Fatalf("reported %+v (%v), want both tests holding no file", reported, err)
	}
}

func TestReportedTestsRefusesATestNameLongerThanALineBeforeReadingIt(t *testing.T) {
	overlong := effectTest + strings.Repeat("A", bufio.MaxScanTokenSize)

	_, err := artifactOf(testSection(overlong, record(effectPath, lineHit))).ReportedCoverage(domain.DeclaredCoverageLimits())

	if !errors.Is(err, bufio.ErrTooLong) || !errors.Is(err, lcov.ErrMalformedArtifact) {
		t.Fatalf("want the line refused as too long, got %v", err)
	}
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
	records := make([]string, 0, count)
	for index := range count {
		records = append(records, record(generatedPath(index), lineHit))
	}
	return artifactOf(testSection(effectTest, records...))
}

func artifactOfTestSections(count int) lcov.Artifact {
	var sections strings.Builder
	for index := range count {
		sections.WriteString(testSection(generatedTest(index), record(effectPath, lineHit)))
	}
	return artifactOf(sections.String())
}

func artifactOfLineRecords(count int) lcov.Artifact {
	return artifactOf(testSection(effectTest, record(effectPath, slices.Repeat([]string{lineHit}, count)...)))
}

func artifactOfTestFilePairs(t *testing.T, count int) lcov.Artifact {
	t.Helper()

	sections := domain.DeclaredCoverageLimits().MaxTestSections
	filesPerTest := (count + sections - 1) / sections
	var plain strings.Builder
	for pair := range count {
		if pair%filesPerTest == 0 {
			plain.WriteString(lines("TN:" + generatedTest(pair/filesPerTest)))
		}
		plain.WriteString(record(generatedPath(pair%filesPerTest), lineHit))
	}
	compressed := gzipped(t, strings.NewReader(plain.String()))
	return lcov.Artifact{DeclaredBytes: int64(compressed.Len()), Body: compressed}
}

func generatedTest(index int) string {
	return fmt.Sprintf("TestGenerated%d", index)
}

func generatedPath(index int) string {
	return fmt.Sprintf("internal/generated/file%d.go", index)
}

func gzipped(t *testing.T, plain io.Reader) *bytes.Buffer {
	t.Helper()

	var compressed bytes.Buffer
	writer, err := gzip.NewWriterLevel(&compressed, gzip.BestSpeed)
	if err != nil {
		t.Fatalf("open a compressor: %v", err)
	}
	if _, err := io.Copy(writer, plain); err != nil {
		t.Fatalf("compress: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("finish compressing: %v", err)
	}
	return &compressed
}

func paddedRecord(totalBytes int64) io.Reader {
	head := testSection(effectTest, lines("SF:"+effectPath))
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
