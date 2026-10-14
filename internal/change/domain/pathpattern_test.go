// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func matches(t *testing.T, pattern, candidate string) bool {
	t.Helper()

	parsed, err := domain.ParsePathPattern(pattern)
	if err != nil {
		t.Fatalf("parse %q: %v", pattern, err)
	}
	return parsed.Matches(candidate)
}

func TestAPatternMatchesOneSegmentAtATime(t *testing.T) {
	matching := map[string]struct {
		pattern, path string
		want          bool
	}{
		"an exact path":                 {"infra/iam.tf", "infra/iam.tf", true},
		"a star within one segment":     {"infra/*.tf", "infra/iam.tf", true},
		"a star does not cross a slash": {"infra/*", "infra/aws/iam.tf", false},
		"a star in the middle":          {"internal/*/middleware.go", "internal/auth/middleware.go", true},
		"a question mark is one byte":   {"infra/ia?.tf", "infra/iam.tf", true},
		"a different directory":         {"infra/*.tf", "charts/iam.tf", false},
		"too few segments":              {"infra/*.tf", "iam.tf", false},
		"too many segments":             {"infra/*.tf", "a/infra/iam.tf", false},
		"a dot is not a wildcard":       {"infra/iam.tf", "infra/iamXtf", false},
	}

	for form, expected := range matching {
		t.Run(form, func(t *testing.T) {
			if got := matches(t, expected.pattern, expected.path); got != expected.want {
				t.Fatalf("%q against %q is %v, want %v",
					expected.pattern, expected.path, got, expected.want)
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
		"two spans":                    {"**/testdata/**", "internal/a/testdata/b/c.json", true},
		"two spans, nothing between":   {"**/testdata/**", "testdata", true},
	}

	for form, expected := range spanning {
		t.Run(form, func(t *testing.T) {
			if got := matches(t, expected.pattern, expected.path); got != expected.want {
				t.Fatalf("%q against %q is %v, want %v",
					expected.pattern, expected.path, got, expected.want)
			}
		})
	}
}

func TestParsePathPatternRefusesWhatItCannotTranslate(t *testing.T) {
	refused := map[string]string{
		"a character class": "infra/[ab]am.tf",
		"a negated class":   "infra/[!a]am.tf",
		"an unclosed class": "infra/[aam.tf",
	}

	for form, pattern := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParsePathPattern(pattern); !errors.Is(err, domain.ErrUnsupportedPathPattern) {
				t.Fatalf("%q gave %v, want ErrUnsupportedPathPattern", pattern, err)
			}
		})
	}
}

func TestManySpansStayAffordable(t *testing.T) {
	const spans = 24
	pattern, err := domain.ParsePathPattern(strings.Repeat("**/", spans) + "x.go")
	if err != nil {
		t.Fatalf("parse a pattern of %d spans: %v", spans, err)
	}
	deep := strings.Repeat("a/", 19) + "y.go"

	started := time.Now()
	if pattern.Matches(deep) {
		t.Fatalf("%d spans matched a path ending y.go", spans)
	}

	if taken := time.Since(started); taken > time.Second {
		t.Fatalf("matching took %v, which is backtracking rather than RE2", taken)
	}
}
