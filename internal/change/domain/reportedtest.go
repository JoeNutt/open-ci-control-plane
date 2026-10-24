// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type ReportedTest struct {
	name  TopLevelTestName
	files []TestedFile
}

func NewReportedTest(name TopLevelTestName, files []TestedFile) ReportedTest {
	return ReportedTest{name: name, files: exercisedOf(onePerPathOf(files))}
}

func exercisedOf(files []TestedFile) []TestedFile {
	var exercised []TestedFile
	for _, file := range files {
		if file.isExercised() {
			exercised = append(exercised, file)
		}
	}
	return exercised
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
