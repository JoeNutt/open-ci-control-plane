// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrRenameWithoutPreviousPath = errors.New("the forge states a rename with no previous path")

const unstatedPreviousPath = ""

type ChangedFile struct {
	path         RepositoryPath
	previousPath RepositoryPath
	lines        ChangedLines
}

func ParseChangedFile(stated StatedFile) (ChangedFile, error) {
	path, err := ParseRepositoryPath(stated.File.Path)
	if err != nil {
		return ChangedFile{}, fmt.Errorf("%q: %w", stated.File.Path, err)
	}
	previous, err := previousPathOf(stated)
	if err != nil {
		return ChangedFile{}, fmt.Errorf("%s: %w", path, err)
	}
	lines, err := ParseChangedLines(stated.Patch)
	if err != nil {
		return ChangedFile{}, fmt.Errorf("%s: %w", path, err)
	}
	return ChangedFile{path: path, previousPath: previous, lines: lines}, nil
}

func previousPathOf(stated StatedFile) (RepositoryPath, error) {
	if stated.File.Action == FileActionRenamed && stated.PreviousPath == unstatedPreviousPath {
		return RepositoryPath{}, ErrRenameWithoutPreviousPath
	}
	previous, _ := ParseRepositoryPath(stated.PreviousPath)
	return previous, nil
}
