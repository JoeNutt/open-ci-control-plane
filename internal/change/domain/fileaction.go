// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

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
