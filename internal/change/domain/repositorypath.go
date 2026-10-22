// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

const repositoryRoot = "."

var ErrPathOutsideRepository = errors.New("path is not inside the repository")

var (
	errAbsolutePath     = fmt.Errorf("%w: it is absolute", ErrPathOutsideRepository)
	errNonCanonicalPath = fmt.Errorf("%w: it is not one canonical file path", ErrPathOutsideRepository)
)

type RepositoryPath struct {
	path string
}

func ParseRepositoryPath(path string) (RepositoryPath, error) {
	if strings.HasPrefix(path, pathSeparator) {
		return RepositoryPath{}, errAbsolutePath
	}
	if !fs.ValidPath(path) || path == repositoryRoot {
		return RepositoryPath{}, errNonCanonicalPath
	}
	return RepositoryPath{path: path}, nil
}
