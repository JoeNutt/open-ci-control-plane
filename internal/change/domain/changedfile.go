// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrRenameWithoutPreviousPath = errors.New("the forge states a rename with no previous path")

var ErrPreviousPathOfNoRenameOrCopy = errors.New("the forge states a previous path for a file that is neither renamed nor copied")

const unstatedPreviousPath = ""

type ChangedFile struct {
	path   RepositoryPath
	action FileAction
	base   basePath
	lines  changedLines
}

func ParseChangedFile(stated StatedFile) (ChangedFile, error) {
	path, err := ParseRepositoryPath(stated.File.Path)
	if err != nil {
		return ChangedFile{}, fmt.Errorf("%q: %w", stated.File.Path, err)
	}
	base, err := basePathOf(stated, path)
	if err != nil {
		return ChangedFile{}, fmt.Errorf("%s: %w", path, err)
	}
	lines, err := linesOf(stated)
	if err != nil {
		return ChangedFile{}, fmt.Errorf("%s: %w", path, err)
	}
	return ChangedFile{path: path, action: stated.File.Action, base: base, lines: lines}, nil
}

func (file ChangedFile) Path() RepositoryPath { return file.path }

func (file ChangedFile) ExistsAtHead() bool { return file.action.ExistsAtHead() }

func linesOf(stated StatedFile) (changedLines, error) {
	if !stated.IsPatchStated {
		return unknownChangedLines(), nil
	}
	return parseChangedLines(stated.Patch)
}

func basePathOf(stated StatedFile, path RepositoryPath) (basePath, error) {
	if stated.File.Action == FileActionRenamed {
		return previousPathOf(stated)
	}
	if stated.PreviousPath != unstatedPreviousPath && !stated.File.Action.mayStateAPreviousPath() {
		return basePath{}, ErrPreviousPathOfNoRenameOrCopy
	}
	if !stated.File.Action.ExistsAtBase() {
		return absentAtBase(), nil
	}
	return atBaseUnder(path), nil
}

func previousPathOf(renamed StatedFile) (basePath, error) {
	if renamed.PreviousPath == unstatedPreviousPath {
		return basePath{}, ErrRenameWithoutPreviousPath
	}
	previous, err := ParseRepositoryPath(renamed.PreviousPath)
	if err != nil {
		return basePath{}, err
	}
	return atBaseUnder(previous), nil
}
