// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

const repositoryRoot = ""

var ErrMalformedRepositoryDirectory = errors.New("directory is not a repository-relative directory")

type RepositoryDirectory struct {
	value string
}

func ParseRepositoryDirectory(stated string) (RepositoryDirectory, error) {
	if stated == repositoryRoot {
		return RepositoryDirectory{value: repositoryRoot}, nil
	}
	if !isRepositoryPath(stated) {
		return RepositoryDirectory{}, fmt.Errorf("%w: %d bytes", ErrMalformedRepositoryDirectory, len(stated))
	}
	return RepositoryDirectory{value: stated}, nil
}

func (directory RepositoryDirectory) String() string { return directory.value }
