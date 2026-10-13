// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrUnknownFileAction = errors.New("unknown file action")

func ParseFileAction(name string) (FileAction, error) {
	if name == "modified" {
		return FileActionModified, nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownFileAction, name)
}
