// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

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
	return TestFunction{name: parsed, lines: lineSpan{first: firstLine, last: lastLine}}, nil
}
