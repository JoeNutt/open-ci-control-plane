// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseCatalogPolicyKeepsItsName(t *testing.T) {
	policy, err := domain.ParseCatalogPolicy(domain.CatalogPolicyStatement{Name: "clean-architecture"})
	if err != nil {
		t.Fatalf("parse catalog policy: %v", err)
	}
	if policy.Name().String() != "clean-architecture" {
		t.Errorf("name is %q, want clean-architecture", policy.Name())
	}
}

func TestParseCatalogPolicyKeepsItsGroupsInOrder(t *testing.T) {
	policy, err := domain.ParseCatalogPolicy(domain.CatalogPolicyStatement{
		Name: "clean-architecture",
		Groups: []domain.GroupStatement{
			{Name: "domain", Order: 0},
			{Name: "adapter", Order: 2},
		},
	})
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
		stated domain.CatalogPolicyStatement
		reason error
	}{
		"a malformed policy name": {
			stated: domain.CatalogPolicyStatement{Name: "clean architecture"},
			reason: domain.ErrMalformedPolicyName,
		},
		"a malformed group name": {
			stated: domain.CatalogPolicyStatement{
				Name:   "clean-architecture",
				Groups: []domain.GroupStatement{{Name: "do main", Order: 0}},
			},
			reason: domain.ErrMalformedGroupName,
		},
	}
	for form, refusal := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParseCatalogPolicy(refusal.stated); !errors.Is(err, refusal.reason) {
				t.Fatalf("%s gave %v, want %v", form, err, refusal.reason)
			}
		})
	}
}
