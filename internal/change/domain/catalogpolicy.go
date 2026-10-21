// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
)

type CatalogPolicyStatement struct {
	Name   string
	Groups []GroupStatement
}

type CatalogPolicy struct {
	name   PolicyName
	groups []PolicyGroup
}

func ParseCatalogPolicy(stated CatalogPolicyStatement) (CatalogPolicy, error) {
	name, err := ParsePolicyName(stated.Name)
	if err != nil {
		return CatalogPolicy{}, fmt.Errorf("catalog policy: %w", err)
	}
	groups, err := policyGroupsOf(stated.Groups)
	if err != nil {
		return CatalogPolicy{}, fmt.Errorf("catalog policy %s: %w", name, err)
	}
	return CatalogPolicy{name: name, groups: groups}, nil
}

func (policy CatalogPolicy) Name() PolicyName { return policy.name }

func (policy CatalogPolicy) Groups() []PolicyGroup { return slices.Clone(policy.groups) }

func policyGroupsOf(stated []GroupStatement) ([]PolicyGroup, error) {
	groups := make([]PolicyGroup, 0, len(stated))
	for _, group := range stated {
		parsed, err := parsePolicyGroup(group)
		if err != nil {
			return nil, err
		}
		groups = append(groups, parsed)
	}
	return groups, nil
}
