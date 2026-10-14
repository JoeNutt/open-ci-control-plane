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
