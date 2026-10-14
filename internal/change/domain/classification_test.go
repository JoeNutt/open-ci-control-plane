// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"fmt"
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

func TestAPathLandsInEveryKindItMatches(t *testing.T) {
	rules := rulesFor(t, map[domain.PathKind][]string{
		domain.PathKindTest:  {"**/*_test.go"},
		domain.PathKindAuthz: {"internal/auth/**"},
	})

	kinds := rules.KindsOf(domain.Paths{"internal/auth/session_test.go"})

	if len(kinds.Test) != 1 {
		t.Errorf("test kind is %v, want the file", kinds.Test)
	}
	if len(kinds.Authz) != 1 {
		t.Errorf("authz kind is %v, want the same file", kinds.Authz)
	}
}

func TestParseClassificationRulesRefusesMorePatternsThanAKindMayHold(t *testing.T) {
	tooMany := make([]string, domain.MaxPatternsPerKind+1)
	for index := range tooMany {
		tooMany[index] = fmt.Sprintf("internal/area%04d/**", index)
	}

	_, err := domain.ParseClassificationRules(map[domain.PathKind][]string{
		domain.PathKindTest: tooMany,
	})

	if !errors.Is(err, domain.ErrTooManyPatternsForKind) {
		t.Fatalf("error is %v, want ErrTooManyPatternsForKind", err)
	}
}

func TestParseClassificationRulesAcceptsAKindAtTheBound(t *testing.T) {
	atBound := make([]string, domain.MaxPatternsPerKind)
	for index := range atBound {
		atBound[index] = fmt.Sprintf("internal/area%04d/**", index)
	}

	if _, err := domain.ParseClassificationRules(map[domain.PathKind][]string{
		domain.PathKindTest: atBound,
	}); err != nil {
		t.Fatalf("a kind at the bound was refused: %v", err)
	}
}

func TestParseDocumentClassifiesEveryChangedPath(t *testing.T) {
	rules := rulesFor(t, map[domain.PathKind][]string{
		domain.PathKindTest: {"**/*_test.go"},
		domain.PathKindIaC:  {"infra/**"},
	})
	bounded := withinBounds(t,
		domain.File{Path: "internal/auth/session_test.go", Action: domain.FileActionAdded},
		domain.File{Path: "infra/iam.tf", Action: domain.FileActionModified},
		domain.File{Path: "README.md", Action: domain.FileActionModified},
	)

	document := domain.ParseDocument(domain.PullRequest{}, bounded, rules)

	if len(document.Diff.Kinds.Test) != 1 {
		t.Errorf("test kind is %v, want the test file", document.Diff.Kinds.Test)
	}
	if len(document.Diff.Kinds.IaC) != 1 {
		t.Errorf("iac kind is %v, want the terraform file", document.Diff.Kinds.IaC)
	}
	if len(document.Diff.Kinds.Route) != 0 {
		t.Errorf("route kind is %v, want nothing", document.Diff.Kinds.Route)
	}
}
