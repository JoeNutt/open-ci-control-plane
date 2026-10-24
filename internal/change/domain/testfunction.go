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

type lineSpan struct {
	first int
	last  int
}

func ParseTestFunction(name string, firstLine, lastLine int) (TestFunction, error) {
	parsed, err := ParseTopLevelTestName(name)
	if err != nil {
		return TestFunction{}, fmt.Errorf("test function: %w", err)
	}
	if firstLine < 1 || lastLine < firstLine {
		return TestFunction{}, fmt.Errorf("%w: lines %d to %d", ErrMalformedLineSpan, firstLine, lastLine)
	}
	return TestFunction{name: parsed, lines: lineSpan{first: firstLine, last: lastLine}}, nil
}
