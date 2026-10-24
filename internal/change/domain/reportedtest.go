// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type ReportedTest struct {
	name  TopLevelTestName
	files []TestedFile
}

func NewReportedTest(name TopLevelTestName, files []TestedFile) ReportedTest {
	var merged []TestedFile
	positionOf := map[RepositoryPath]int{}
	for _, file := range files {
		position, isHeld := positionOf[file.path]
		if !isHeld {
			positionOf[file.path] = len(merged)
			merged = append(merged, file)
			continue
		}
		merged[position] = NewTestedFile(file.path, append(merged[position].hitLines, file.hitLines...))
	}
	return ReportedTest{name: name, files: merged}
}
