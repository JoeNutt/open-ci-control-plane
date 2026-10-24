// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrMalformedLineSpan = errors.New("the span is not a run of lines in a file")

type TestFunction struct {
	name  TopLevelTestName
	lines lineSpan
}

func ParseTestFunction(name string, firstLine, lastLine int) (TestFunction, error) {
	parsed, err := ParseTopLevelTestName(name)
	if err != nil {
		return TestFunction{}, err
	}
	lines, err := parseLineSpan(firstLine, lastLine)
	if err != nil {
		return TestFunction{}, err
	}
	return TestFunction{name: parsed, lines: lines}, nil
}

type lineSpan struct {
	first int
	last  int
}

func parseLineSpan(first, last int) (lineSpan, error) {
	if first < firstLineNumber || last < first {
		return lineSpan{}, fmt.Errorf("%w: lines %d to %d", ErrMalformedLineSpan, first, last)
	}
	return lineSpan{first: first, last: last}, nil
}

func (span lineSpan) holds(line int) bool { return span.first <= line && line <= span.last }
