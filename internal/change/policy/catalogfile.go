// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var policyRuleByWord = map[string]domain.PolicyRule{
	"direction": domain.DirectionRule,
}

var policyDirectionByWord = map[string]domain.PolicyDirection{
	"lower_order_may_not_import_higher": domain.LowerOrderMayNotImportHigher,
}

type catalogDocument struct {
	SchemaVersion *int                `yaml:"schema_version"`
	Name          string              `yaml:"name"`
	Rule          string              `yaml:"rule"`
	Direction     string              `yaml:"direction"`
	Groups        []catalogGroupEntry `yaml:"groups"`
}

type catalogGroupEntry struct {
	Name  string `yaml:"name"`
	Order *int   `yaml:"order"`
}

func ParseCatalogFile(content []byte) (domain.CatalogPolicy, error) {
	parsed, err := catalogPolicyIn(content)
	if err != nil {
		return domain.CatalogPolicy{}, fmt.Errorf("catalog file: %w", err)
	}
	return parsed, nil
}

func catalogPolicyIn(content []byte) (domain.CatalogPolicy, error) {
	document, err := decodedStrictly[catalogDocument](content)
	if err != nil {
		return domain.CatalogPolicy{}, err
	}
	statement, err := catalogStatementOf(document)
	if err != nil {
		return domain.CatalogPolicy{}, err
	}
	return domain.ParseCatalogPolicy(statement)
}

func catalogStatementOf(document catalogDocument) (domain.CatalogPolicyStatement, error) {
	if err := refuseUnreadSchemaVersion(document.SchemaVersion); err != nil {
		return domain.CatalogPolicyStatement{}, err
	}
	rule, isKnownRule := policyRuleByWord[document.Rule]
	if !isKnownRule {
		return domain.CatalogPolicyStatement{}, fmt.Errorf("%w: %d bytes", domain.ErrUnknownPolicyRule, len(document.Rule))
	}
	direction, isKnownDirection := policyDirectionByWord[document.Direction]
	if !isKnownDirection {
		return domain.CatalogPolicyStatement{}, fmt.Errorf("%w: %d bytes",
			domain.ErrUnknownPolicyDirection, len(document.Direction))
	}
	groups, err := groupStatementsOf(document.Groups)
	if err != nil {
		return domain.CatalogPolicyStatement{}, err
	}
	return domain.CatalogPolicyStatement{
		Name:      document.Name,
		Rule:      rule,
		Direction: direction,
		Groups:    groups,
	}, nil
}

func groupStatementsOf(entries []catalogGroupEntry) ([]domain.GroupStatement, error) {
	groups := make([]domain.GroupStatement, 0, len(entries))
	for _, entry := range entries {
		if entry.Order == nil {
			return nil, fmt.Errorf("%w: order of a group", ErrMissingKey)
		}
		groups = append(groups, domain.GroupStatement{Name: entry.Name, Order: *entry.Order})
	}
	return groups, nil
}
