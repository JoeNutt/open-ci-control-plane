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
	position, isSeen := union.positionOf[file.Path()]
	if !isSeen {
		union.positionOf[file.Path()] = len(union.files)
		union.files = append(union.files, file)
		return
	}
	isExercised := union.files[position].IsExercised() || file.IsExercised()
	union.files[position] = domain.NewReportedFile(file.Path(), isExercised)
}

func (union fileUnion) count() int { return len(union.files) }
