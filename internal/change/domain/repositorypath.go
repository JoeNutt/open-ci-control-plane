// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

const rootElement = "."

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
	if !isCanonicalSlashPath(path) || !isRepositoryPath(path) {
		return RepositoryPath{}, errNonCanonicalPath
	}
	return RepositoryPath{path: path}, nil
}

func isCanonicalSlashPath(path string) bool {
	return fs.ValidPath(path) && path != rootElement
}
