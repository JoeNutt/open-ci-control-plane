// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
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
