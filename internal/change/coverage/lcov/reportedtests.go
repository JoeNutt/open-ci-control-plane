// SPDX-License-Identifier: GPL-3.0-or-later

package lcov

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	tagSeparator       = ":"
	lineDataSeparator  = ","
	endOfRecord        = "end_of_record"
	lineNumberSlot     = 0
	executionCountSlot = 1
	lineNumberBits     = 32
	firstSourceLine    = 1
	firstLineNumber    = 1
	decimal            = 10
	countBits          = 64
	gzipMagic          = "\x1f\x8b"
)

var ErrMalformedArtifact = fmt.Errorf("%w as LCOV", domain.ErrMalformedCoverageReport)

var (
	errUnknownTag         = fmt.Errorf("%w: unknown line tag", ErrMalformedArtifact)
	errOutOfOrder         = fmt.Errorf("%w: line is out of place for an open or closed record", ErrMalformedArtifact)
	errUnreadableLineData = fmt.Errorf("%w: line data is not a line number and an execution count", ErrMalformedArtifact)
	errTruncatedRecord    = fmt.Errorf("%w: input ends inside a record", ErrMalformedArtifact)
	errNoRecord           = fmt.Errorf("%w: input reports no file", ErrMalformedArtifact)
	errOutsideTestSection = fmt.Errorf("%w: a file record precedes any TN section", ErrMalformedArtifact)
	errNotATopLevelTest   = fmt.Errorf("%w: a TN section names no top-level test", ErrMalformedArtifact)
	errTestNamedTwice     = fmt.Errorf("%w: a test is named by two TN sections", ErrMalformedArtifact)
)

type lineReading struct {
	isInsideRecord bool
	step           func(*recordReading, string) error
}

var readingByTag = map[string]lineReading{
	"TN":        {isInsideRecord: false, step: (*recordReading).openTestSection},
	"SF":        {isInsideRecord: false, step: (*recordReading).openRecord},
	"DA":        {isInsideRecord: true, step: (*recordReading).countExecutions},
	"LF":        {isInsideRecord: true, step: carryingNothing},
	"LH":        {isInsideRecord: true, step: carryingNothing},
	endOfRecord: {isInsideRecord: true, step: (*recordReading).closeRecord},
}

type recordReading struct {
	reported     reportedPaths
	currentPath  domain.RepositoryPath
	isRecordOpen bool
	testSections boundedCount
	testNames    map[domain.TopLevelTestName]bool
	lineRecords  boundedCount
	limits       domain.CoverageLimits
	tests        []domain.ReportedTest
	currentTest  domain.TopLevelTestName
	currentFiles []domain.TestedFile
	currentHits  []uint32
}

type Artifact struct {
	DeclaredBytes int64
	Body          io.Reader
}

func (artifact Artifact) ReportedTests(limits domain.CoverageLimits) ([]domain.ReportedTest, error) {
	reading, err := artifact.completedReading(limits)
	if err != nil {
		return nil, err
	}
	return reading.tests, nil
}

func (artifact Artifact) completedReading(limits domain.CoverageLimits) (*recordReading, error) {
	if breach := limits.CompressedBytesBreach(artifact.DeclaredBytes); breach != nil {
		return nil, breach
	}
	compressed := &boundedReader{source: artifact.Body, breachOf: limits.CompressedBytesBreach}
	plain, err := plainBodyOf(compressed)
	if err != nil {
		return nil, err
	}
	decompressed := &boundedReader{source: plain, breachOf: limits.DecompressedBytesBreach}
	reading := readingWithin(limits)
	if err := reading.takeAll(bufio.NewScanner(decompressed)); err != nil {
		return nil, err
	}
	if err := reading.complete(); err != nil {
		return nil, err
	}
	return &reading, nil
}

func readingWithin(limits domain.CoverageLimits) recordReading {
	return recordReading{
		reported:     reportedPaths{},
		testSections: boundedCount{breachOf: limits.TestSectionsBreach},
		testNames:    map[domain.TopLevelTestName]bool{},
		lineRecords:  boundedCount{breachOf: limits.LineRecordsBreach},
		limits:       limits,
	}
}

func plainBodyOf(body io.Reader) (io.Reader, error) {
	buffered := bufio.NewReader(body)
	leading, err := buffered.Peek(len(gzipMagic))
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, readFailureOf(err)
	}
	if !bytes.Equal(leading, []byte(gzipMagic)) {
		return buffered, nil
	}
	decompressed, err := gzip.NewReader(buffered)
	if err != nil {
		return nil, readFailureOf(err)
	}
	return decompressed, nil
}

func (reading *recordReading) takeAll(lines *bufio.Scanner) error {
	for lineNumber := firstLineNumber; lines.Scan(); lineNumber++ {
		if err := reading.take(lines.Text()); err != nil {
			return fmt.Errorf("line %d: %w", lineNumber, err)
		}
	}
	return readFailureOf(lines.Err())
}

func readFailureOf(err error) error {
	if err == nil || errors.Is(err, domain.ErrCoverageBoundExceeded) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrMalformedArtifact, err)
}

func (reading *recordReading) take(line string) error {
	tag, value, _ := strings.Cut(line, tagSeparator)
	known, isKnown := readingByTag[tag]
	if !isKnown {
		return errUnknownTag
	}
	if known.isInsideRecord != reading.isRecordOpen {
		return errOutOfOrder
	}
	return known.step(reading, value)
}

func (reading *recordReading) openTestSection(name string) error {
	reading.closeTest()
	if breach := reading.testSections.countOneMore(); breach != nil {
		return breach
	}
	parsed, err := domain.ParseTopLevelTestName(name)
	if err != nil {
		return fmt.Errorf("%w: %w", errNotATopLevelTest, err)
	}
	if reading.testNames[parsed] {
		return errTestNamedTwice
	}
	reading.testNames[parsed] = true
	reading.currentTest = parsed
	return nil
}

func (reading *recordReading) openRecord(path string) error {
	if !reading.testSections.hasCounted() {
		return errOutsideTestSection
	}
	parsed, err := domain.ParseRepositoryPath(path)
	if err != nil {
		return err
	}
	if breach := reading.limits.FileEntriesBreach(reading.reported.entriesWith(parsed)); breach != nil {
		return breach
	}
	reading.currentPath = parsed
	reading.currentHits = nil
	reading.isRecordOpen = true
	return nil
}

func (reading *recordReading) countExecutions(lineData string) error {
	if breach := reading.lineRecords.countOneMore(); breach != nil {
		return breach
	}
	fields := strings.Split(lineData, lineDataSeparator)
	if len(fields) <= executionCountSlot {
		return errUnreadableLineData
	}
	line, err := sourceLineOf(fields[lineNumberSlot])
	if err != nil {
		return err
	}
	executions, err := strconv.ParseUint(fields[executionCountSlot], decimal, countBits)
	if err != nil {
		return errUnreadableLineData
	}
	if executions == 0 {
		return nil
	}
	reading.currentHits = append(reading.currentHits, line)
	return nil
}

func sourceLineOf(stated string) (uint32, error) {
	line, err := strconv.ParseUint(stated, decimal, lineNumberBits)
	if err != nil || line < firstSourceLine {
		return 0, errUnreadableLineData
	}
	return uint32(line), nil
}

func (reading *recordReading) closeRecord(string) error {
	reading.reported[reading.currentPath] = true
	reading.currentFiles = append(reading.currentFiles, domain.NewTestedFile(reading.currentPath, reading.currentHits))
	reading.isRecordOpen = false
	return nil
}

type boundedReader struct {
	source    io.Reader
	bytesRead int64
	breachOf  func(bytes int64) error
}

func (bounded *boundedReader) Read(buffer []byte) (int, error) {
	count, err := bounded.source.Read(buffer)
	bounded.bytesRead += int64(count)
	if breach := bounded.breachOf(bounded.bytesRead); breach != nil {
		return 0, breach
	}
	return count, err
}

type boundedCount struct {
	counted  int
	breachOf func(count int) error
}

func (bounded *boundedCount) countOneMore() error {
	withThisOne := bounded.counted + 1
	if breach := bounded.breachOf(withThisOne); breach != nil {
		return breach
	}
	bounded.counted = withThisOne
	return nil
}

func (bounded boundedCount) hasCounted() bool { return bounded.counted > 0 }

func (reading *recordReading) closeTest() {
	if !reading.testSections.hasCounted() {
		return
	}
	reading.tests = append(reading.tests, domain.NewReportedTest(reading.currentTest, reading.currentFiles))
	reading.currentFiles = nil
}

func carryingNothing(*recordReading, string) error { return nil }

func (reading *recordReading) complete() error {
	if reading.isRecordOpen {
		return errTruncatedRecord
	}
	if len(reading.reported) == 0 {
		return errNoRecord
	}
	reading.closeTest()
	return nil
}
