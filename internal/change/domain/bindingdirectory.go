// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
)

var ErrMalformedRepositoryDirectory = errors.New("directory is not a repository-relative directory")

type RepositoryDirectory struct {
	value string
}

func ParseRepositoryDirectory(stated string) (RepositoryDirectory, error) {
	if stated == "" {
		return RepositoryDirectory{}, nil
	}
	if len(stated) > DeclaredLimits().MaxPathBytes {
		return RepositoryDirectory{}, fmt.Errorf("%w: %d bytes", ErrMalformedRepositoryDirectory, len(stated))
	}
	for segment := range strings.SplitSeq(stated, pathSeparator) {
		if !importPathSegmentRule.admits(segment) {
			return RepositoryDirectory{}, fmt.Errorf("%w: %d bytes", ErrMalformedRepositoryDirectory, len(stated))
		}
	}
	return RepositoryDirectory{value: stated}, nil
}

func (directory RepositoryDirectory) String() string { return directory.value }
