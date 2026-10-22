// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestEveryFamilyTheDesignDeclaresIsNamed(t *testing.T) {
	declared := map[string]domain.PolicyFamily{
		"structural": domain.PolicyFamilyStructural,
		"security":   domain.PolicyFamilySecurity,
		"quality":    domain.PolicyFamilyQuality,
		"routing":    domain.PolicyFamilyRouting,
	}

	for name, family := range declared {
		t.Run(name, func(t *testing.T) {
			parsed, err := domain.ParsePolicyFamily(name)
			if err != nil {
				t.Fatalf("parse %q: %v", name, err)
			}
			if parsed != family {
				t.Errorf("%q parsed to %q", name, parsed)
			}
		})
	}
}

func TestParsePolicyFamilyRefusesAnythingElse(t *testing.T) {
	for _, name := range []string{"", "convention", "SECURITY", "architecture"} {
		t.Run(name, func(t *testing.T) {
			if _, err := domain.ParsePolicyFamily(name); !errors.Is(err, domain.ErrUnknownPolicyFamily) {
				t.Fatalf("%q gave %v, want ErrUnknownPolicyFamily", name, err)
			}
		})
	}
}

func TestAFamilyNamesItsOwnDecisionQuery(t *testing.T) {
	queries := map[domain.PolicyFamily]string{
		domain.PolicyFamilyStructural: "data.ci.structural.decision",
		domain.PolicyFamilySecurity:   "data.ci.security.decision",
		domain.PolicyFamilyQuality:    "data.ci.quality.decision",
		domain.PolicyFamilyRouting:    "data.ci.routing.decision",
	}

	for family, want := range queries {
		t.Run(string(family), func(t *testing.T) {
			if family.DecisionQuery() != want {
				t.Fatalf("query is %q, want %q", family.DecisionQuery(), want)
			}
		})
	}
}

func TestEachFamilyCarriesTheFailurePostureTheDesignStates(t *testing.T) {
	postures := map[domain.PolicyFamily]domain.FailurePosture{
		domain.PolicyFamilySecurity:   domain.FailurePostureClosed,
		domain.PolicyFamilyRouting:    domain.FailurePostureOpen,
		domain.PolicyFamilyQuality:    domain.FailurePostureOpen,
		domain.PolicyFamilyStructural: domain.FailurePostureClosed,
	}

	for family, want := range postures {
		t.Run(string(family), func(t *testing.T) {
			if family.FailurePosture() != want {
				t.Fatalf("posture is %q, want %q", family.FailurePosture(), want)
			}
		})
	}
}

func TestAnUndeclaredFamilyFailsClosed(t *testing.T) {
	undeclared := domain.PolicyFamily("nonsense")

	if posture := undeclared.FailurePosture(); posture != domain.FailurePostureClosed {
		t.Fatalf("posture is %q, want %q", posture, domain.FailurePostureClosed)
	}
}
