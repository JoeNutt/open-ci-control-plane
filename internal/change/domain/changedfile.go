// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrRenameWithoutPreviousPath = errors.New("the forge states a rename with no previous path")

const unstatedPreviousPath = ""

type ChangedFile struct {
	path   RepositoryPath
	action FileAction
	base   basePath
	lines  ChangedLines
}

type basePath struct {
	path     RepositoryPath
	isAtBase bool
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

func linesOf(stated StatedFile) (ChangedLines, error) {
	if !stated.IsPatchStated {
		return UnknownChangedLines(), nil
	}
	return ParseChangedLines(stated.Patch)
}

func basePathOf(stated StatedFile, path RepositoryPath) (basePath, error) {
	if stated.PreviousPath != unstatedPreviousPath {
		previous, err := ParseRepositoryPath(stated.PreviousPath)
		if err != nil {
			return basePath{}, err
		}
		return atBaseUnder(previous), nil
	}
	if stated.File.Action == FileActionRenamed {
		return basePath{}, ErrRenameWithoutPreviousPath
	}
	if !stated.File.Action.ExistsAtBase() {
		return absentAtBase(), nil
	}
	return atBaseUnder(path), nil
}

func absentAtBase() basePath { return basePath{} }

func atBaseUnder(path RepositoryPath) basePath { return basePath{path: path, isAtBase: true} }
