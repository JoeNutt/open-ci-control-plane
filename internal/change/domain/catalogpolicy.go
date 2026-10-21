// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

type CatalogPolicyStatement struct {
	Name string
}

type CatalogPolicy struct {
	name PolicyName
}

func ParseCatalogPolicy(stated CatalogPolicyStatement) (CatalogPolicy, error) {
	name, err := ParsePolicyName(stated.Name)
	if err != nil {
		return CatalogPolicy{}, fmt.Errorf("catalog policy: %w", err)
	}
	return CatalogPolicy{name: name}, nil
}

func (policy CatalogPolicy) Name() PolicyName { return policy.name }
