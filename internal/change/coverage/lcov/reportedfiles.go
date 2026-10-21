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
)

var ErrMalformedArtifact = errors.New("artifact is not parseable as LCOV")

var readingByTag = map[string]func(*recordReading, string) error{
	"TN":        carryingNothing,
	"SF":        (*recordReading).openRecord,
	"DA":        (*recordReading).countExecutions,
	"LF":        carryingNothing,
	"LH":        carryingNothing,
	endOfRecord: (*recordReading).closeRecord,
}

type recordReading struct {
	reported []domain.ReportedFile
	current  domain.ReportedFile
}

func ReportedFilesIn(artifact io.Reader) ([]domain.ReportedFile, error) {
	var reading recordReading
	lines := bufio.NewScanner(artifact)
	for lineNumber := firstLineNumber; lines.Scan(); lineNumber++ {
		if err := reading.take(lines.Text()); err != nil {
			return nil, fmt.Errorf("%w: line %d: %w", ErrMalformedArtifact, lineNumber, err)
		}
	}
	if err := lines.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedArtifact, err)
	}
	return reading.reported, nil
}

var errUnknownTag = errors.New("unknown line tag")

func (reading *recordReading) take(line string) error {
	tag, value, _ := strings.Cut(line, tagSeparator)
	step, isKnown := readingByTag[tag]
	if !isKnown {
		return errUnknownTag
	}
	return step(reading, value)
}

func (reading *recordReading) openRecord(path string) error {
	reading.current = domain.ReportedFile{Path: path}
	return nil
}

var errUnreadableLineData = errors.New("line data is not a line number and an execution count")

func (reading *recordReading) countExecutions(lineData string) error {
	fields := strings.Split(lineData, lineDataSeparator)
	if len(fields) <= executionCountSlot {
		return errUnreadableLineData
	}
	executions, err := strconv.Atoi(fields[executionCountSlot])
	if err != nil {
		return errUnreadableLineData
	}
	reading.current.IsExercised = reading.current.IsExercised || executions > 0
	return nil
}

func (reading *recordReading) closeRecord(string) error {
	reading.reported = append(reading.reported, reading.current)
	return nil
}

func carryingNothing(*recordReading, string) error { return nil }
