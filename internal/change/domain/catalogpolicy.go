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
	return CatalogPolicy{name: name, groups: policyGroupsOf(stated.Groups)}, nil
}

func (policy CatalogPolicy) Name() PolicyName { return policy.name }

func (policy CatalogPolicy) Groups() []PolicyGroup { return slices.Clone(policy.groups) }

func policyGroupsOf(stated []GroupStatement) []PolicyGroup {
	groups := make([]PolicyGroup, 0, len(stated))
	for _, group := range stated {
		name, _ := ParseGroupName(group.Name)
		groups = append(groups, PolicyGroup{name: name, order: group.Order})
	}
	return groups
}
