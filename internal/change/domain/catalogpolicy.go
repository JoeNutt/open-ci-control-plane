// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

var ErrDuplicateGroup = errors.New("catalog policy declares one group twice")

type CatalogPolicyStatement struct {
	Name      string
	Rule      PolicyRule
	Direction PolicyDirection
	Groups    []GroupStatement
}

type CatalogPolicy struct {
	name      PolicyName
	rule      PolicyRule
	direction PolicyDirection
	groups    []PolicyGroup
}

func ParseCatalogPolicy(stated CatalogPolicyStatement) (CatalogPolicy, error) {
	name, err := ParsePolicyName(stated.Name)
	if err != nil {
		return CatalogPolicy{}, fmt.Errorf("catalog policy: %w", err)
	}
	if !stated.Rule.isShipped() {
		return CatalogPolicy{}, fmt.Errorf("catalog policy %s: %w: %d", name, ErrUnknownPolicyRule, stated.Rule)
	}
	if !stated.Direction.isShipped() {
		return CatalogPolicy{}, fmt.Errorf("catalog policy %s: %w: %d", name, ErrUnknownPolicyDirection, stated.Direction)
	}
	groups, err := policyGroupsOf(stated.Groups)
	if err != nil {
		return CatalogPolicy{}, fmt.Errorf("catalog policy %s: %w", name, err)
	}
	return CatalogPolicy{name: name, rule: stated.Rule, direction: stated.Direction, groups: groups}, nil
}

func (policy CatalogPolicy) Direction() PolicyDirection { return policy.direction }

func (policy CatalogPolicy) Name() PolicyName { return policy.name }

func (policy CatalogPolicy) Rule() PolicyRule { return policy.rule }

func (policy CatalogPolicy) Groups() []PolicyGroup { return slices.Clone(policy.groups) }

func policyGroupsOf(stated []GroupStatement) ([]PolicyGroup, error) {
	groups := make([]PolicyGroup, 0, len(stated))
	for _, group := range stated {
		parsed, err := parsePolicyGroup(group)
		if err != nil {
			return nil, err
		}
		if slices.ContainsFunc(groups, parsed.sharesNameWith) {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateGroup, parsed.name)
		}
		groups = append(groups, parsed)
	}
	return groups, nil
}
