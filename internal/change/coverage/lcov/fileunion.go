// SPDX-License-Identifier: GPL-3.0-or-later

package lcov

import "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"

type fileUnion struct {
	files      []domain.ReportedFile
	positionOf map[domain.RepositoryPath]int
}

func emptyFileUnion() fileUnion {
	return fileUnion{positionOf: map[domain.RepositoryPath]int{}}
}

func (union *fileUnion) include(file domain.ReportedFile) {
	position, isHeld := union.positionOf[file.Path()]
	if !isHeld {
		union.positionOf[file.Path()] = len(union.files)
		union.files = append(union.files, file)
		return
	}
	isExercised := union.files[position].IsExercised() || file.IsExercised()
	union.files[position] = domain.NewReportedFile(file.Path(), isExercised)
}

func (union fileUnion) reportedFiles() []domain.ReportedFile { return union.files }

func (union fileUnion) entriesWith(path domain.RepositoryPath) int {
	if union.holds(path) {
		return union.entries()
	}
	return union.entries() + 1
}

func (union fileUnion) entries() int { return len(union.files) }

func (union fileUnion) holds(path domain.RepositoryPath) bool {
	_, isHeld := union.positionOf[path]
	return isHeld
}
