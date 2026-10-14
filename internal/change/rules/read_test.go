// SPDX-License-Identifier: GPL-3.0-or-later

package rules_test

import (
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/rules"
)

func TestReadFromGivesRulesThatClassify(t *testing.T) {
	stated := `{
		"kinds": {
			"test": ["**/*_test.go"],
			"iac":  ["infra/**", "**/*.tf"]
		}
	}`

	read, err := rules.ReadFrom(strings.NewReader(stated))
	if err != nil {
		t.Fatalf("read rules: %v", err)
	}

	kinds := read.KindsOf(domain.Paths{
		"internal/auth/session_test.go",
		"infra/iam.tf",
		"README.md",
	})
	if len(kinds.Test) != 1 {
		t.Errorf("test kind is %v, want the test file", kinds.Test)
	}
	if len(kinds.IaC) != 1 {
		t.Errorf("iac kind is %v, want the terraform file", kinds.IaC)
	}
}

func TestReadFromRefusesADocumentItCannotTrust(t *testing.T) {
	refused := map[string]string{
		"a misspelled top-level key":          `{"kind": {"test": ["**/*_test.go"]}}`,
		"a kind the schema does not declare":  `{"kinds": {"tests": ["**/*_test.go"]}}`,
		"a pattern that cannot be translated": `{"kinds": {"test": ["infra/[ab].go"]}}`,
		"more patterns than a kind may hold": `{"kinds": {"test": [` +
			strings.TrimSuffix(strings.Repeat(`"a/**",`, domain.MaxPatternsPerKind+1), ",") + `]}}`,
		"not an object at all": `["**/*_test.go"]`,
		"truncated":            `{"kinds": {"test": [`,
	}

	for form, stated := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := rules.ReadFrom(strings.NewReader(stated)); err == nil {
				t.Fatalf("%s was accepted", form)
			}
		})
	}
}
