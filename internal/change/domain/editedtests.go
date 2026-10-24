// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "slices"

type EditedTests struct {
	path  RepositoryPath
	names []TopLevelTestName
}

func EditedTestsIn(file ChangedFile, functions []TestFunction) EditedTests {
	var names []TopLevelTestName
	for _, function := range functions {
		if !file.lines.isKnown || slices.ContainsFunc(file.lines.head, function.lines.holds) {
			names = append(names, function.name)
		}
	}
	return EditedTests{path: file.path, names: names}
}
