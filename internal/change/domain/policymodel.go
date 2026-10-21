// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"slices"
)

var ErrUnknownCatalogPolicy = errors.New("binding names a policy absent from the catalog")

type PolicyModel struct {
	catalog  []CatalogPolicy
	bindings []Binding
}

func ParsePolicyModel(catalog []CatalogPolicy, bindings []Binding) (PolicyModel, error) {
	for _, binding := range bindings {
		for _, bound := range binding.policies {
			if !slices.ContainsFunc(catalog, func(policy CatalogPolicy) bool { return policy.name == bound.use }) {
				return PolicyModel{}, ErrUnknownCatalogPolicy
			}
		}
	}
	return PolicyModel{catalog: slices.Clone(catalog), bindings: slices.Clone(bindings)}, nil
}

func (model PolicyModel) Catalog() []CatalogPolicy { return slices.Clone(model.catalog) }

func (model PolicyModel) Bindings() []Binding { return slices.Clone(model.bindings) }
