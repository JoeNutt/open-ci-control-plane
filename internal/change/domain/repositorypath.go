// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
)

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
	if !isRepositoryPath(path) {
		return RepositoryPath{}, errNonCanonicalPath
	}
	return RepositoryPath{path: path}, nil
}

func (path RepositoryPath) String() string { return path.path }

func (path RepositoryPath) compare(other RepositoryPath) int {
	return strings.Compare(path.path, other.path)
}

func (path RepositoryPath) Segments() []string { return strings.Split(path.path, pathSeparator) }
