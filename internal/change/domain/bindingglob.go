// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"strings"
)

const directoryPatternSuffix = pathSeparator + spanningSegment

type BoundGlob struct {
	directory RepositoryDirectory
	pattern   PathPattern
}

func parseBoundGlob(stated string) (BoundGlob, error) {
	written, isDirectoryPattern := strings.CutSuffix(stated, directoryPatternSuffix)
	if !isDirectoryPattern || written == repositoryRoot {
		return BoundGlob{}, fmt.Errorf("%w: not a directory followed by %s", ErrUnsupportedPathPattern, directoryPatternSuffix)
	}
	directory, err := ParseRepositoryDirectory(written)
	if err != nil {
		return BoundGlob{}, fmt.Errorf("%w: %w", ErrUnsupportedPathPattern, err)
	}
	pattern, err := ParsePathPattern(stated)
	if err != nil {
		return BoundGlob{}, err
	}
	return BoundGlob{directory: directory, pattern: pattern}, nil
}

func (bound BoundGlob) Glob() string { return bound.directory.String() + directoryPatternSuffix }

func (bound BoundGlob) Directory() RepositoryDirectory { return bound.directory }

func (bound BoundGlob) Matches(candidate string) bool { return bound.pattern.Matches(candidate) }

func (bound BoundGlob) translatesLike(other BoundGlob) bool {
	return bound.directory == other.directory
}
