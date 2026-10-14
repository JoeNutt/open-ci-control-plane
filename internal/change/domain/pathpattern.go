// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

var ErrMalformedPathPattern = errors.New("path pattern is malformed")

type PathPattern struct {
	segments []string
}

func ParsePathPattern(pattern string) (PathPattern, error) {
	segments := strings.Split(pattern, pathSeparator)
	for _, segment := range segments {
		if _, err := path.Match(segment, ""); err != nil {
			return PathPattern{}, fmt.Errorf("%w: %q: %w", ErrMalformedPathPattern, pattern, err)
		}
	}
	return PathPattern{segments: segments}, nil
}

func (pattern PathPattern) Matches(candidate string) bool {
	return segmentsMatch(pattern.segments, strings.Split(candidate, pathSeparator))
}

func segmentsMatch(patterns, segments []string) bool {
	if len(patterns) != len(segments) {
		return false
	}
	for index, segment := range segments {
		matched, err := path.Match(patterns[index], segment)
		if err != nil || !matched {
			return false
		}
	}
	return true
}
