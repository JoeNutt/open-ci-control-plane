// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParsePolicyModelKeepsTheCatalogAndBindings(t *testing.T) {
	model, err := domain.ParsePolicyModel(
		[]domain.CatalogPolicy{parsedCatalogPolicy(t, cleanArchitecture())},
		[]domain.Binding{parsedBinding(t, thisRepositorysBinding())},
	)
	if err != nil {
		t.Fatalf("parse policy model: %v", err)
	}
	if len(model.Catalog()) != 1 || len(model.Bindings()) != 1 {
		t.Errorf("model holds %d catalog policies and %d bindings, want one of each",
			len(model.Catalog()), len(model.Bindings()))
	}
}

func TestParsePolicyModelRefusesABindingNamingAnUnknownPolicy(t *testing.T) {
	stated := thisRepositorysBinding()
	stated.Policies[0].Use = "mvvm"

	_, err := domain.ParsePolicyModel(
		[]domain.CatalogPolicy{parsedCatalogPolicy(t, cleanArchitecture())},
		[]domain.Binding{parsedBinding(t, stated)},
	)
	if !errors.Is(err, domain.ErrUnknownCatalogPolicy) {
		t.Fatalf("a binding naming mvvm gave %v, want ErrUnknownCatalogPolicy", err)
	}
}

func parsedCatalogPolicy(t *testing.T, stated domain.CatalogPolicyStatement) domain.CatalogPolicy {
	t.Helper()

	policy, err := domain.ParseCatalogPolicy(stated)
	if err != nil {
		t.Fatalf("parse catalog policy: %v", err)
	}
	return policy
}

func parsedBinding(t *testing.T, stated domain.BindingStatement) domain.Binding {
	t.Helper()

	binding, err := domain.ParseBinding(stated)
	if err != nil {
		t.Fatalf("parse binding: %v", err)
	}
	return binding
}
