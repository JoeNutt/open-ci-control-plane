// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type StatedFile struct {
	File          File
	PreviousPath  string
	Patch         string
	IsPatchStated bool
}

func (stated StatedFile) DocumentFile() File {
	file := stated.File
	file.PatchBytes = len(stated.Patch)
	return file
}
