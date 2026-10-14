// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

const spanningSegment = "**"

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
	if len(patterns) == 0 {
		return len(segments) == 0
	}
	if patterns[0] == spanningSegment {
		return spanMatches(patterns[1:], segments)
	}
	if len(segments) == 0 {
		return false
	}
	matched, err := path.Match(patterns[0], segments[0])
	if err != nil || !matched {
		return false
	}
	return segmentsMatch(patterns[1:], segments[1:])
}

func spanMatches(beyond, segments []string) bool {
	for spanned := 0; spanned <= len(segments); spanned++ {
		if segmentsMatch(beyond, segments[spanned:]) {
			return true
		}
	}
	return false
}
