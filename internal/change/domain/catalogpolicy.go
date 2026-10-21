// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type CatalogPolicyStatement struct {
	Name string
}

type CatalogPolicy struct {
	name PolicyName
}

func ParseCatalogPolicy(stated CatalogPolicyStatement) (CatalogPolicy, error) {
	name, err := ParsePolicyName(stated.Name)
	return CatalogPolicy{name: name}, err
}

func (policy CatalogPolicy) Name() PolicyName { return policy.name }
