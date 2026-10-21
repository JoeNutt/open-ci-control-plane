// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrMalformedPolicyFile = errors.New("policy file is not YAML of the declared shape")

var policyRuleByWord = map[string]domain.PolicyRule{
	"direction": domain.DirectionRule,
}

var policyDirectionByWord = map[string]domain.PolicyDirection{
	"lower_order_may_not_import_higher": domain.LowerOrderMayNotImportHigher,
}

type catalogDocument struct {
	SchemaVersion int                 `yaml:"schema_version"`
	Name          string              `yaml:"name"`
	Rule          string              `yaml:"rule"`
	Direction     string              `yaml:"direction"`
	Groups        []catalogGroupEntry `yaml:"groups"`
}

type catalogGroupEntry struct {
	Name  string `yaml:"name"`
	Order int    `yaml:"order"`
}

func ParseCatalogFile(content []byte) (domain.CatalogPolicy, error) {
	var document catalogDocument
	if err := yaml.Unmarshal(content, &document); err != nil {
		return domain.CatalogPolicy{}, ErrMalformedPolicyFile
	}

	statement, err := catalogStatementOf(document)
	if err != nil {
		return domain.CatalogPolicy{}, err
	}
	return domain.ParseCatalogPolicy(statement)
}

func catalogStatementOf(document catalogDocument) (domain.CatalogPolicyStatement, error) {
	rule, isKnownRule := policyRuleByWord[document.Rule]
	if !isKnownRule {
		return domain.CatalogPolicyStatement{}, fmt.Errorf("%w: %d bytes", domain.ErrUnknownPolicyRule, len(document.Rule))
	}
	direction, isKnownDirection := policyDirectionByWord[document.Direction]
	if !isKnownDirection {
		return domain.CatalogPolicyStatement{}, fmt.Errorf("%w: %d bytes",
			domain.ErrUnknownPolicyDirection, len(document.Direction))
	}
	return domain.CatalogPolicyStatement{
		Name:      document.Name,
		Rule:      rule,
		Direction: direction,
		Groups:    groupStatementsOf(document.Groups),
	}, nil
}

func groupStatementsOf(entries []catalogGroupEntry) []domain.GroupStatement {
	groups := make([]domain.GroupStatement, 0, len(entries))
	for _, entry := range entries {
		groups = append(groups, domain.GroupStatement{Name: entry.Name, Order: entry.Order})
	}
	return groups
}
