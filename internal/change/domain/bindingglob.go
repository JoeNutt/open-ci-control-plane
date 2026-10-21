// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"strings"
)

type BoundGlob struct {
	glob    string
	pattern PathPattern
}

func parseBoundGlob(stated string) (BoundGlob, error) {
	directory, isDirectoryPattern := strings.CutSuffix(stated, "/**")
	if !isDirectoryPattern || directory == "" {
		return BoundGlob{}, fmt.Errorf("%w: not a directory pattern", ErrUnsupportedPathPattern)
	}
	if _, err := ParseRepositoryDirectory(directory); err != nil {
		return BoundGlob{}, fmt.Errorf("%w: %w", ErrUnsupportedPathPattern, err)
	}
	pattern, err := ParsePathPattern(stated)
	if err != nil {
		return BoundGlob{}, err
	}
	return BoundGlob{glob: stated, pattern: pattern}, nil
}

func (bound BoundGlob) Glob() string { return bound.glob }

func (bound BoundGlob) Matches(candidate string) bool { return bound.pattern.Matches(candidate) }

func (bound BoundGlob) translatesLike(other BoundGlob) bool {
	return bound.pattern.translatesLike(other.pattern)
}
