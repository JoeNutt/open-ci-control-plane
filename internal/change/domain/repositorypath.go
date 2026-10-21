// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
)

var ErrPathOutsideRepository = errors.New("path is not inside the repository")

type RepositoryPath struct {
	path string
}

func ParseRepositoryPath(path string) (RepositoryPath, error) {
	if strings.HasPrefix(path, pathSeparator) {
		return RepositoryPath{}, fmt.Errorf("%w: it is absolute", ErrPathOutsideRepository)
	}
	return RepositoryPath{path: path}, nil
}
