// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
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
