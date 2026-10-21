// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"go.yaml.in/yaml/v3"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type catalogDocument struct {
	SchemaVersion int    `yaml:"schema_version"`
	Name          string `yaml:"name"`
	Rule          string `yaml:"rule"`
	Direction     string `yaml:"direction"`
	Groups        []struct {
		Name  string `yaml:"name"`
		Order int    `yaml:"order"`
	} `yaml:"groups"`
}

func ParseCatalogFile(content []byte) (domain.CatalogPolicy, error) {
	var doc catalogDocument
	yaml.Unmarshal(content, &doc)
	var groups []domain.GroupStatement
	for _, g := range doc.Groups {
		groups = append(groups, domain.GroupStatement{Name: g.Name, Order: g.Order})
	}
	return domain.ParseCatalogPolicy(domain.CatalogPolicyStatement{
		Name:      doc.Name,
		Rule:      domain.DirectionRule,
		Direction: domain.LowerOrderMayNotImportHigher,
		Groups:    groups,
	})
}
