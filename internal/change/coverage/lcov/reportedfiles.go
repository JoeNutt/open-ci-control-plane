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
	executionCountSlot = 1
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
	errUnnamedTest        = fmt.Errorf("%w: a TN section names no test", ErrMalformedArtifact)
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
	reported           fileUnion
	currentPath        domain.RepositoryPath
	isCurrentExercised bool
	isRecordOpen       bool
	hasTestSection     bool
	limits             domain.CoverageLimits
}

type Artifact struct {
	DeclaredBytes int64
	Body          io.Reader
}

func (artifact Artifact) ReportedFiles(limits domain.CoverageLimits) ([]domain.ReportedFile, error) {
	if breach := limits.CompressedBytesBreach(artifact.DeclaredBytes); breach != nil {
		return nil, breach
	}
	compressed := &boundedReader{source: artifact.Body, breachOf: limits.CompressedBytesBreach}
	plain, err := plainBodyOf(compressed)
	if err != nil {
		return nil, err
	}
	decompressed := &boundedReader{source: plain, breachOf: limits.DecompressedBytesBreach}
	reading := recordReading{reported: emptyFileUnion(), limits: limits}
	if err := reading.takeAll(bufio.NewScanner(decompressed)); err != nil {
		return nil, err
	}
	return reading.completed()
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
	if name == "" {
		return errUnnamedTest
	}
	reading.hasTestSection = true
	return nil
}

func (reading *recordReading) openRecord(path string) error {
	if !reading.hasTestSection {
		return errOutsideTestSection
	}
	entriesWithThisOne := reading.reported.count() + 1
	if breach := reading.limits.FileEntriesBreach(entriesWithThisOne); breach != nil {
		return breach
	}
	parsed, err := domain.ParseRepositoryPath(path)
	if err != nil {
		return err
	}
	reading.currentPath = parsed
	reading.isCurrentExercised = false
	reading.isRecordOpen = true
	return nil
}

func (reading *recordReading) countExecutions(lineData string) error {
	fields := strings.Split(lineData, lineDataSeparator)
	if len(fields) <= executionCountSlot {
		return errUnreadableLineData
	}
	executions, err := strconv.ParseUint(fields[executionCountSlot], decimal, countBits)
	if err != nil {
		return errUnreadableLineData
	}
	reading.isCurrentExercised = reading.isCurrentExercised || executions > 0
	return nil
}

func (reading *recordReading) closeRecord(string) error {
	reading.reported.include(domain.NewReportedFile(reading.currentPath, reading.isCurrentExercised))
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

func carryingNothing(*recordReading, string) error { return nil }

func (reading *recordReading) completed() ([]domain.ReportedFile, error) {
	if reading.isRecordOpen {
		return nil, errTruncatedRecord
	}
	if reading.reported.count() == 0 {
		return nil, errNoRecord
	}
	return reading.reported.files, nil
}
