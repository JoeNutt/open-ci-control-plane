// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type EditedTests struct {
	path  RepositoryPath
	names []TopLevelTestName
}

func EditedTestsIn(file ChangedFile, functions []TestFunction) EditedTests {
	var names []TopLevelTestName
	for _, function := range functions {
		for _, line := range file.lines.head {
			if line >= function.lines.first && line <= function.lines.last {
				names = append(names, function.name)
				break
			}
		}
	}
	return EditedTests{path: file.path, names: names}
}
