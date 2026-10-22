// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

var ErrUnknownCatalogPolicy = errors.New("binding names a policy absent from the catalog")

var ErrDuplicateCatalogPolicy = errors.New("catalog holds two policies with one name")

var ErrDuplicateBinding = errors.New("policy repository holds two bindings for one repository")

var ErrUndeclaredGroup = errors.New("binding names a group its catalog policy does not declare")

type PolicyModel struct {
	catalog  []CatalogPolicy
	bindings []Binding
}

func ParsePolicyModel(catalog []CatalogPolicy, bindings []Binding) (PolicyModel, error) {
	if err := duplicateCatalogPolicyIn(catalog); err != nil {
		return PolicyModel{}, err
	}
	if err := duplicateBindingIn(bindings); err != nil {
		return PolicyModel{}, err
	}
	if err := unresolvedBindingIn(bindings, catalog); err != nil {
		return PolicyModel{}, err
	}
	return PolicyModel{catalog: slices.Clone(catalog), bindings: slices.Clone(bindings)}, nil
}

func (model PolicyModel) Catalog() []CatalogPolicy { return slices.Clone(model.catalog) }

func (model PolicyModel) Bindings() []Binding { return slices.Clone(model.bindings) }

func (model PolicyModel) BindingOf(repository GovernedRepository) (Binding, error) {
	bound := slices.IndexFunc(model.bindings, func(binding Binding) bool { return binding.repository == repository })
	if bound < 0 {
		return Binding{}, fmt.Errorf("%w: %s", ErrNoBinding, repository)
	}
	return model.bindings[bound], nil
}

func duplicateCatalogPolicyIn(catalog []CatalogPolicy) error {
	for index, policy := range catalog {
		if slices.ContainsFunc(catalog[:index], policy.sharesNameWith) {
			return fmt.Errorf("%w: %s", ErrDuplicateCatalogPolicy, policy.name)
		}
	}
	return nil
}

func duplicateBindingIn(bindings []Binding) error {
	for index, binding := range bindings {
		if slices.ContainsFunc(bindings[:index], binding.governsSameRepositoryAs) {
			return fmt.Errorf("%w: %s", ErrDuplicateBinding, binding.repository)
		}
	}
	return nil
}

func unresolvedBindingIn(bindings []Binding, catalog []CatalogPolicy) error {
	for _, binding := range bindings {
		if err := unresolvedPolicyOf(binding, catalog); err != nil {
			return fmt.Errorf("binding for %s: %w", binding.repository, err)
		}
	}
	return nil
}

func (policy CatalogPolicy) sharesNameWith(other CatalogPolicy) bool {
	return policy.name == other.name
}

func (binding Binding) governsSameRepositoryAs(other Binding) bool {
	return binding.repository == other.repository
}

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
