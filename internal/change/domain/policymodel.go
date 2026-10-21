// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

var ErrUnknownCatalogPolicy = errors.New("binding names a policy absent from the catalog")

var ErrDuplicateBinding = errors.New("policy repository holds two bindings for one repository")

var ErrUndeclaredGroup = errors.New("binding names a group its catalog policy does not declare")

type PolicyModel struct {
	catalog  []CatalogPolicy
	bindings []Binding
}

func ParsePolicyModel(catalog []CatalogPolicy, bindings []Binding) (PolicyModel, error) {
	for index, binding := range bindings {
		for _, earlier := range bindings[:index] {
			if earlier.repository == binding.repository {
				return PolicyModel{}, ErrDuplicateBinding
			}
		}
		if err := unresolvedPolicyOf(binding, catalog); err != nil {
			return PolicyModel{}, fmt.Errorf("binding for %s: %w", binding.repository, err)
		}
	}
	return PolicyModel{catalog: slices.Clone(catalog), bindings: slices.Clone(bindings)}, nil
}

func (model PolicyModel) Catalog() []CatalogPolicy { return slices.Clone(model.catalog) }

func (model PolicyModel) Bindings() []Binding { return slices.Clone(model.bindings) }

func unresolvedPolicyOf(binding Binding, catalog []CatalogPolicy) error {
	for _, bound := range binding.policies {
		used := slices.IndexFunc(catalog, bound.isUseOf)
		if used < 0 {
			return fmt.Errorf("%w: %s", ErrUnknownCatalogPolicy, bound.use)
		}
		if err := undeclaredGroupOf(bound, catalog[used]); err != nil {
			return err
		}
	}
	return nil
}

func undeclaredGroupOf(bound BoundPolicy, catalogued CatalogPolicy) error {
	for _, paths := range bound.groups {
		if !catalogued.declares(paths.group) {
			return fmt.Errorf("%w: %s in %s", ErrUndeclaredGroup, paths.group, catalogued.name)
		}
	}
	return nil
}

func (policy CatalogPolicy) declares(group GroupName) bool {
	return slices.ContainsFunc(policy.groups, func(declared PolicyGroup) bool { return declared.name == group })
}

func (policy BoundPolicy) isUseOf(catalogued CatalogPolicy) bool {
	return policy.use == catalogued.name
}
