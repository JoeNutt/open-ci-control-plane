// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseCatalogPolicyKeepsWhatItStates(t *testing.T) {
	policy, err := domain.ParseCatalogPolicy(cleanArchitecture())
	if err != nil {
		t.Fatalf("parse catalog policy: %v", err)
	}
	if policy.Name().String() != "clean-architecture" {
		t.Errorf("name is %q, want clean-architecture", policy.Name())
	}
	if policy.Rule() != domain.DirectionRule {
		t.Errorf("rule is %v, want the direction rule", policy.Rule())
	}
	if policy.Direction() != domain.LowerOrderMayNotImportHigher {
		t.Errorf("direction is %v, want lower order may not import higher", policy.Direction())
	}
}

func TestParseCatalogPolicyKeepsItsGroupsInOrder(t *testing.T) {
	policy, err := domain.ParseCatalogPolicy(cleanArchitecture())
	if err != nil {
		t.Fatalf("parse catalog policy: %v", err)
	}

	groups := policy.Groups()
	if len(groups) != 2 {
		t.Fatalf("groups are %v, want two", groups)
	}
	if groups[1].Name().String() != "adapter" || groups[1].Order() != 2 {
		t.Errorf("second group is %s at %d, want adapter at 2", groups[1].Name(), groups[1].Order())
	}
}

func TestParseCatalogPolicyRefusesAPolicyItCannotTrust(t *testing.T) {
	refused := map[string]struct {
		stated func(*domain.CatalogPolicyStatement)
		reason error
	}{
		"a malformed policy name": {
			stated: func(s *domain.CatalogPolicyStatement) { s.Name = "clean architecture" },
			reason: domain.ErrMalformedPolicyName,
		},
		"a malformed group name": {
			stated: func(s *domain.CatalogPolicyStatement) { s.Groups[0].Name = "do main" },
			reason: domain.ErrMalformedGroupName,
		},
		"a negative group order": {
			stated: func(s *domain.CatalogPolicyStatement) { s.Groups[0].Order = -1 },
			reason: domain.ErrNegativeGroupOrder,
		},
		"two groups with one name": {
			stated: func(s *domain.CatalogPolicyStatement) { s.Groups[1].Name = s.Groups[0].Name },
			reason: domain.ErrDuplicateGroup,
		},
		"no rule": {
			stated: func(s *domain.CatalogPolicyStatement) { s.Rule = domain.PolicyRule(0) },
			reason: domain.ErrUnknownPolicyRule,
		},
		"no direction": {
			stated: func(s *domain.CatalogPolicyStatement) { s.Direction = domain.PolicyDirection(0) },
			reason: domain.ErrUnknownPolicyDirection,
		},
	}
	for form, refusal := range refused {
		t.Run(form, func(t *testing.T) {
			stated := cleanArchitecture()
			refusal.stated(&stated)
			if _, err := domain.ParseCatalogPolicy(stated); !errors.Is(err, refusal.reason) {
				t.Fatalf("%s gave %v, want %v", form, err, refusal.reason)
			}
		})
	}
}

func cleanArchitecture() domain.CatalogPolicyStatement {
	return domain.CatalogPolicyStatement{
		Name:      "clean-architecture",
		Rule:      domain.DirectionRule,
		Direction: domain.LowerOrderMayNotImportHigher,
		Groups:    []domain.GroupStatement{{Name: "domain", Order: 0}, {Name: "adapter", Order: 2}},
	}
}
