// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

type Limits struct {
	MaxChangedFiles int
}

type BoundedFiles struct {
	files []File
}

var ErrTooManyChangedFiles = errors.New("change exceeds the changed file bound")

func ParseBoundedFiles(files []File, limits Limits) (BoundedFiles, error) {
	if len(files) > limits.MaxChangedFiles {
		return BoundedFiles{}, fmt.Errorf("%w: %d exceeds %d",
			ErrTooManyChangedFiles, len(files), limits.MaxChangedFiles)
	}
	return BoundedFiles{files: files}, nil
}
