// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "slices"

type ReportedTest struct {
	name  TopLevelTestName
	files []TestedFile
}

func NewReportedTest(name TopLevelTestName, files []TestedFile) ReportedTest {
	joined := onePerPathOf(files)
	joined = slices.DeleteFunc(joined, func(file TestedFile) bool { return !file.isExercised() })
	return ReportedTest{name: name, files: joined}
}

func onePerPathOf(files []TestedFile) []TestedFile {
	var joined []TestedFile
	positionOf := map[RepositoryPath]int{}
	for _, file := range files {
		position, isHeld := positionOf[file.path]
		if isHeld {
			joined[position] = joined[position].joinedWith(file)
			continue
		}
		positionOf[file.path] = len(joined)
		joined = append(joined, file)
	}
	return joined
}
