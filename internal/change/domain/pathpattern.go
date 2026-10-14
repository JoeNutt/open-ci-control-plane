// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const spanningSegment = "**"

var ErrUnsupportedPathPattern = errors.New("path pattern uses syntax this project does not translate")

type PathPattern struct {
	expression *regexp.Regexp
}

func ParsePathPattern(pattern string) (PathPattern, error) {
	if strings.ContainsAny(pattern, "[]") {
		return PathPattern{}, fmt.Errorf("%w: %q: character classes", ErrUnsupportedPathPattern, pattern)
	}
	expression, err := regexp.Compile(expressionFor(pattern))
	if err != nil {
		return PathPattern{}, fmt.Errorf("%w: %q: %w", ErrUnsupportedPathPattern, pattern, err)
	}
	return PathPattern{expression: expression}, nil
}

func (pattern PathPattern) Matches(candidate string) bool {
	return pattern.expression.MatchString(candidate)
}

func expressionFor(pattern string) string {
	segments := strings.Split(pattern, pathSeparator)
	var built strings.Builder
	built.WriteString("^")
	for index, segment := range segments {
		switch {
		case segment == spanningSegment && index == len(segments)-1:
			built.WriteString(`(?:[^/]*(?:/[^/]*)*)?`)
		case segment == spanningSegment:
			built.WriteString(`(?:[^/]*/)*`)
		default:
			if index > 0 && segments[index-1] != spanningSegment {
				built.WriteString(pathSeparator)
			}
			built.WriteString(expressionForSegment(segment))
		}
	}
	built.WriteString("$")
	return built.String()
}

func expressionForSegment(segment string) string {
	var built strings.Builder
	for _, character := range segment {
		switch character {
		case '*':
			built.WriteString(`[^/]*`)
		case '?':
			built.WriteString(`[^/]`)
		default:
			built.WriteString(regexp.QuoteMeta(string(character)))
		}
	}
	return built.String()
}
