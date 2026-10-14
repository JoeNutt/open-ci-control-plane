// SPDX-License-Identifier: GPL-3.0-or-later

package rules_test

import (
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/rules"
)

const written = `
# A kind is a top-level key. Everything under it is a path pattern.
test:
  - "**/*_test.go"
  - "**/testdata/**"

# Terraform and anything under infra/ counts as infrastructure.
iac:
  - "infra/**"
  - "**/*.tf"
`

func TestReadFromGivesRulesThatClassify(t *testing.T) {
	read, err := rules.ReadFrom(strings.NewReader(written))
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
		"a kind the schema does not declare":  "tests:\n  - \"**/*_test.go\"\n",
		"a pattern that cannot be translated": "test:\n  - \"infra/[ab].go\"\n",
		"a kind holding a single string":      "test: \"**/*_test.go\"\n",
		"a sequence where a mapping belongs":  "- \"**/*_test.go\"\n",
		"more patterns than a kind may hold": "test:\n" +
			strings.Repeat("  - \"a/**\"\n", domain.MaxPatternsPerKind+1),
		"not yaml at all": "test:\n  - \"a\"\n :\t- bad\n",
	}

	for form, stated := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := rules.ReadFrom(strings.NewReader(stated)); err == nil {
				t.Fatalf("%s was accepted", form)
			}
		})
	}
}
