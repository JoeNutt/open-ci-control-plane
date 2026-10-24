// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type EditedTests struct {
	path                 RepositoryPath
	names                []TopLevelTestName
	isEditingBeyondTests bool
}

func EditedTestsIn(file ChangedFile, functions []TestFunction) EditedTests {
	var names []TopLevelTestName
	for _, function := range functions {
		if file.lines.mayTouchHead(function.lines) {
			names = append(names, function.name)
		}
	}
	isBeyond := !file.lines.isKnown
	for _, line := range file.lines.head {
		inside := false
		for _, function := range functions {
			if function.lines.holds(line) {
				inside = true
			}
		}
		if !inside {
			isBeyond = true
		}
	}
	return EditedTests{path: file.path, names: names, isEditingBeyondTests: isBeyond}
}
