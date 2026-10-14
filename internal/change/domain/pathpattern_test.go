// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAPatternWithoutSpanningMatchesSegmentAtATime(t *testing.T) {
	matching := map[string]struct {
		pattern, path string
		want          bool
	}{
		"an exact path":                 {"infra/iam.tf", "infra/iam.tf", true},
		"a star within one segment":     {"infra/*.tf", "infra/iam.tf", true},
		"a star does not cross a slash": {"infra/*", "infra/aws/iam.tf", false},
		"a star in the middle":          {"internal/*/middleware.go", "internal/auth/middleware.go", true},
		"a different directory":         {"infra/*.tf", "charts/iam.tf", false},
		"too few segments":              {"infra/*.tf", "iam.tf", false},
		"too many segments":             {"infra/*.tf", "a/infra/iam.tf", false},
	}

	for form, expected := range matching {
		t.Run(form, func(t *testing.T) {
			pattern, err := domain.ParsePathPattern(expected.pattern)
			if err != nil {
				t.Fatalf("parse %q: %v", expected.pattern, err)
			}

			if pattern.Matches(expected.path) != expected.want {
				t.Fatalf("%q against %q is %v, want %v",
					expected.pattern, expected.path,
					pattern.Matches(expected.path), expected.want)
			}
		})
	}
}

func TestASpanningPatternCrossesAnyNumberOfSegments(t *testing.T) {
	spanning := map[string]struct {
		pattern, path string
		want          bool
	}{
		"spans several segments":       {"**/*_test.go", "internal/auth/session_test.go", true},
		"spans one segment":            {"**/*_test.go", "auth/session_test.go", true},
		"spans no segments at all":     {"**/*_test.go", "session_test.go", true},
		"spans in the middle":          {"internal/**/middleware.go", "internal/a/b/middleware.go", true},
		"spans nothing in the middle":  {"internal/**/middleware.go", "internal/middleware.go", true},
		"still honours the tail":       {"**/*_test.go", "internal/auth/session.go", false},
		"still honours the head":       {"internal/**/middleware.go", "infra/a/middleware.go", false},
		"everything under a directory": {"infra/**", "infra/aws/iam.tf", true},
		"the directory itself":         {"infra/**", "infra", true},
		"a bare span matches anything": {"**", "a/b/c.go", true},
	}

	for form, expected := range spanning {
		t.Run(form, func(t *testing.T) {
			pattern, err := domain.ParsePathPattern(expected.pattern)
			if err != nil {
				t.Fatalf("parse %q: %v", expected.pattern, err)
			}

			if pattern.Matches(expected.path) != expected.want {
				t.Fatalf("%q against %q is %v, want %v",
					expected.pattern, expected.path,
					pattern.Matches(expected.path), expected.want)
			}
		})
	}
}

func TestParsePathPatternRefusesMoreSpansThanItWillEvaluate(t *testing.T) {
	bounded := map[string]struct {
		pattern string
		refused bool
	}{
		"no span":                {"internal/auth/*.go", false},
		"one span":               {"**/*_test.go", false},
		"two spans":              {"**/testdata/**", false},
		"two spans, apart":       {"**/internal/**/*_test.go", false},
		"three spans":            {"**/a/**/b/**/*.go", true},
		"a span for every level": {strings.Repeat("**/", 12) + "x.go", true},
	}

	for form, expected := range bounded {
		t.Run(form, func(t *testing.T) {
			_, err := domain.ParsePathPattern(expected.pattern)

			if expected.refused && !errors.Is(err, domain.ErrPathPatternTooBroad) {
				t.Fatalf("%q gave %v, want ErrPathPatternTooBroad", expected.pattern, err)
			}
			if !expected.refused && err != nil {
				t.Fatalf("%q was refused: %v", expected.pattern, err)
			}
		})
	}
}
