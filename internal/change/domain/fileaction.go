// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"slices"
)

type FileAction string

const (
	FileActionAdded    FileAction = "added"
	FileActionModified FileAction = "modified"
	FileActionRemoved  FileAction = "removed"
	FileActionRenamed  FileAction = "renamed"
	FileActionCopied   FileAction = "copied"
	FileActionChanged  FileAction = "changed"
)

var declaredFileActions = []FileAction{
	FileActionAdded,
	FileActionModified,
	FileActionRemoved,
	FileActionRenamed,
	FileActionCopied,
	FileActionChanged,
}

var ErrUnknownFileAction = errors.New("unknown file action")

func ParseFileAction(name string) (FileAction, error) {
	return parseDeclaration(name, declaredFileActions, ErrUnknownFileAction)
}

func (action FileAction) ExistsAtHead() bool { return action.presence().atHead }

func (action FileAction) ExistsAtBase() bool { return action.presence().atBase }

var actionsLeavingTheBasePath = []FileAction{FileActionRemoved, FileActionRenamed}

func (action FileAction) leavesItsBasePath() bool {
	return slices.Contains(actionsLeavingTheBasePath, action)
}

func (action FileAction) mayStateAPreviousPath() bool {
	return action == FileActionRenamed || action == FileActionCopied
}

type presenceAtEnds struct {
	atBase bool
	atHead bool
}

var presenceAfter = map[FileAction]presenceAtEnds{
	FileActionAdded:    {atBase: false, atHead: true},
	FileActionModified: {atBase: true, atHead: true},
	FileActionRemoved:  {atBase: true, atHead: false},
	FileActionRenamed:  {atBase: false, atHead: true},
	FileActionCopied:   {atBase: false, atHead: true},
	FileActionChanged:  {atBase: true, atHead: true},
}

var presenceAfterAnUndeclaredAction = presenceAtEnds{atBase: false, atHead: true}

func (action FileAction) presence() presenceAtEnds {
	present, declared := presenceAfter[action]
	if !declared {
		return presenceAfterAnUndeclaredAction
	}
	return present
}
