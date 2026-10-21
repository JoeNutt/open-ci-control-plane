// SPDX-License-Identifier: GPL-3.0-or-later

package lcov

import (
	"bufio"
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
)

var ErrMalformedArtifact = errors.New("artifact is not parseable as LCOV")

var (
	errUnknownTag         = fmt.Errorf("%w: unknown line tag", ErrMalformedArtifact)
	errOutOfOrder         = fmt.Errorf("%w: line is out of place for an open or closed record", ErrMalformedArtifact)
	errUnreadableLineData = fmt.Errorf("%w: line data is not a line number and an execution count", ErrMalformedArtifact)
	errTruncatedRecord    = fmt.Errorf("%w: input ends inside a record", ErrMalformedArtifact)
	errNoRecord           = fmt.Errorf("%w: input reports no file", ErrMalformedArtifact)
)

type lineReading struct {
	isInsideRecord bool
	step           func(*recordReading, string) error
}

var readingByTag = map[string]lineReading{
	"TN":        {isInsideRecord: false, step: carryingNothing},
	"SF":        {isInsideRecord: false, step: (*recordReading).openRecord},
	"DA":        {isInsideRecord: true, step: (*recordReading).countExecutions},
	"LF":        {isInsideRecord: true, step: carryingNothing},
	"LH":        {isInsideRecord: true, step: carryingNothing},
	endOfRecord: {isInsideRecord: true, step: (*recordReading).closeRecord},
}

type recordReading struct {
	reported           []domain.ReportedFile
	currentPath        domain.RepositoryPath
	isCurrentExercised bool
	isRecordOpen       bool
}

type Artifact struct {
	DeclaredBytes int64
	Body          io.Reader
}

func ReportedFilesIn(artifact Artifact, limits domain.CoverageLimits) ([]domain.ReportedFile, error) {
	if breach := limits.CompressedBytesBreach(artifact.DeclaredBytes); breach != nil {
		return nil, breach
	}
	var reading recordReading
	lines := bufio.NewScanner(&boundedReader{source: artifact.Body, breachOf: limits.CompressedBytesBreach})
	for lineNumber := firstLineNumber; lines.Scan(); lineNumber++ {
		if err := reading.take(lines.Text()); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNumber, err)
		}
	}
	if err := lines.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedArtifact, err)
	}
	return reading.completed()
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

func (reading *recordReading) openRecord(path string) error {
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
	reading.reported = append(reading.reported,
		domain.NewReportedFile(reading.currentPath, reading.isCurrentExercised))
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
	if len(reading.reported) == 0 {
		return nil, errNoRecord
	}
	return reading.reported, nil
}
