// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParsePolicyNameKeepsTheName(t *testing.T) {
	name, err := domain.ParsePolicyName("clean-architecture")
	if err != nil {
		t.Fatalf("parse policy name: %v", err)
	}
	if name.String() != "clean-architecture" {
		t.Errorf("name is %q, want clean-architecture", name)
	}
}

func TestParsePolicyNameRefusesWhatIsNotAnIdentifier(t *testing.T) {
	refused := map[string]string{
		"empty":               "",
		"a space":             "clean architecture",
		"a leading hyphen":    "-clean",
		"a path separator":    "clean/architecture",
		"a non-ASCII letter":  "cléan",
		"one byte over bound": strings.Repeat("a", domain.MaxPolicyNameBytes+1),
	}
	for form, stated := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParsePolicyName(stated); !errors.Is(err, domain.ErrMalformedPolicyName) {
				t.Fatalf("%s gave %v, want ErrMalformedPolicyName", form, err)
			}
		})
	}
}

func TestParsePolicyNameAdmitsANameAtTheBound(t *testing.T) {
	if _, err := domain.ParsePolicyName(strings.Repeat("a", domain.MaxPolicyNameBytes)); err != nil {
		t.Fatalf("a name at the bound was refused: %v", err)
	}
}
