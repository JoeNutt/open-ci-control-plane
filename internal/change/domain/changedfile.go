// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrRenameWithoutPreviousPath = errors.New("the forge states a rename with no previous path")

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
	if stated.File.Action == FileActionRenamed && stated.PreviousPath == "" {
		return ChangedFile{}, fmt.Errorf("%s: %w", path, ErrRenameWithoutPreviousPath)
	}
	previous, _ := ParseRepositoryPath(stated.PreviousPath)
	lines, err := ParseChangedLines(stated.Patch)
	if err != nil {
		return ChangedFile{}, fmt.Errorf("%s: %w", path, err)
	}
	return ChangedFile{path: path, previousPath: previous, lines: lines}, nil
}
