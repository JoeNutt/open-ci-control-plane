// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

type CatalogPolicyStatement struct {
	Name   string
	Groups []GroupStatement
}

type GroupStatement struct {
	Name  string
	Order int
}

type CatalogPolicy struct {
	name   PolicyName
	groups []PolicyGroup
}

type PolicyGroup struct {
	name  GroupName
	order int
}

func ParseCatalogPolicy(stated CatalogPolicyStatement) (CatalogPolicy, error) {
	name, err := ParsePolicyName(stated.Name)
	if err != nil {
		return CatalogPolicy{}, fmt.Errorf("catalog policy: %w", err)
	}
	var groups []PolicyGroup
	for _, g := range stated.Groups {
		n, _ := ParseGroupName(g.Name)
		groups = append(groups, PolicyGroup{name: n, order: g.Order})
	}
	return CatalogPolicy{name: name, groups: groups}, nil
}

func (policy CatalogPolicy) Name() PolicyName { return policy.name }

func (policy CatalogPolicy) Groups() []PolicyGroup { return policy.groups }

func (group PolicyGroup) Name() GroupName { return group.name }

func (group PolicyGroup) Order() int { return group.order }
