// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

// FileAction is what a change did to one file. The set is closed.
type FileAction string

// The file actions the schema declares.
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

// ErrUnknownFileAction reports a file action the schema does not declare.
var ErrUnknownFileAction = errors.New("unknown file action")

// ParseFileAction converts a forge's file action into the declared FileAction it names.
// It returns ErrUnknownFileAction, naming the value, when the schema declares no such
// action.
func ParseFileAction(name string) (FileAction, error) {
	action := FileAction(name)
	if slices.Contains(declaredFileActions, action) {
		return action, nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownFileAction, name)
}
