// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func rulesFor(t *testing.T, stated map[domain.PathKind][]string) domain.ClassificationRules {
	t.Helper()

	rules, err := domain.ParseClassificationRules(stated)
	if err != nil {
		t.Fatalf("parse classification rules: %v", err)
	}
	return rules
}

func TestAPathLandsInTheKindWhosePatternItMatches(t *testing.T) {
	rules := rulesFor(t, map[domain.PathKind][]string{
		domain.PathKindTest: {"**/*_test.go"},
	})

	kinds := rules.KindsOf(domain.Paths{
		"internal/auth/session_test.go",
		"internal/auth/session.go",
	})

	if len(kinds.Test) != 1 || kinds.Test[0] != "internal/auth/session_test.go" {
		t.Fatalf("test kind is %v, want the one test file", kinds.Test)
	}
}
