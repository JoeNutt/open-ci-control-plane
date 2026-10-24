// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type TestFunction struct {
	name      TopLevelTestName
	firstLine int
	lastLine  int
}

func ParseTestFunction(name string, firstLine, lastLine int) (TestFunction, error) {
	parsed, err := ParseTopLevelTestName(name)
	if err != nil {
		return TestFunction{}, err
	}
	return TestFunction{name: parsed, firstLine: firstLine, lastLine: lastLine}, nil
}
