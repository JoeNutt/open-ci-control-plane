// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

// ErrUnknownFileAction reports a file action the schema does not declare.
var ErrUnknownFileAction = errors.New("unknown file action")

// ParseFileAction converts a forge's file action into the declared FileAction it names.
// It returns ErrUnknownFileAction, naming the value, when the schema declares no such
// action.
func ParseFileAction(name string) (FileAction, error) {
	if name == "modified" {
		return FileActionModified, nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownFileAction, name)
}
