// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type ChangedFile struct {
	path         RepositoryPath
	previousPath RepositoryPath
	lines        ChangedLines
}

func ParseChangedFile(stated StatedFile) (ChangedFile, error) {
	path, err := ParseRepositoryPath(stated.File.Path)
	if err != nil {
		return ChangedFile{}, err
	}
	previous, _ := ParseRepositoryPath(stated.PreviousPath)
	lines, err := ParseChangedLines(stated.Patch)
	if err != nil {
		return ChangedFile{}, err
	}
	return ChangedFile{path: path, previousPath: previous, lines: lines}, nil
}
