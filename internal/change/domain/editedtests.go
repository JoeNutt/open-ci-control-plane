// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "slices"

type EditedTests struct {
	path                 RepositoryPath
	names                []TopLevelTestName
	isEditingBeyondTests bool
}

func EditedTestsIn(file ChangedFile, functions []TestFunction) EditedTests {
	var names []TopLevelTestName
	for _, function := range functions {
		if file.base.path != file.path || file.lines.mayTouchHead(function.lines) {
			names = append(names, function.name)
		}
	}
	return EditedTests{path: file.path, names: names, isEditingBeyondTests: file.lines.mayTouchHeadOutside(functions)}
}

func (lines changedLines) mayTouchHeadOutside(functions []TestFunction) bool {
	isInsideATest := func(line int) bool {
		return slices.ContainsFunc(functions, func(function TestFunction) bool { return function.lines.holds(line) })
	}
	isOutsideEveryTest := func(line int) bool { return !isInsideATest(line) }
	return !lines.isKnown || slices.ContainsFunc(lines.head, isOutsideEveryTest)
}
