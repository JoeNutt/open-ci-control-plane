// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type ReportedFile struct {
	path        RepositoryPath
	isExercised bool
}

func NewReportedFile(path RepositoryPath, isExercised bool) ReportedFile {
	return ReportedFile{path: path, isExercised: isExercised}
}

func (reported ReportedFile) Path() RepositoryPath { return reported.path }

func (reported ReportedFile) IsExercised() bool { return reported.isExercised }
