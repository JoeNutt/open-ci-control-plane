// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type policyData struct {
	Catalog  map[string]catalogData `json:"catalog"`
	Bindings map[string]bindingData `json:"bindings"`
}

type catalogData struct {
	Rule      string      `json:"rule"`
	Direction string      `json:"direction"`
	Groups    []groupData `json:"groups"`
}

type groupData struct {
	Name  string `json:"name"`
	Order int    `json:"order"`
}

type bindingData struct {
	Policies []boundPolicyData `json:"policies"`
}

type boundPolicyData struct {
	Use         string              `json:"use"`
	ImportRoots []importRootData    `json:"import_roots"`
	Paths       map[string][]string `json:"paths"`
}

type importRootData struct {
	Prefix string `json:"prefix"`
	Path   string `json:"path"`
}

func policyDataOf(model domain.PolicyModel) (policyData, error) {
	data := policyData{Catalog: map[string]catalogData{}, Bindings: map[string]bindingData{}}
	for _, catalogued := range model.Catalog() {
		served, err := catalogDataOf(catalogued)
		if err != nil {
			return policyData{}, err
		}
		data.Catalog[catalogued.Name().String()] = served
	}
	for _, binding := range model.Bindings() {
		data.Bindings[binding.Repository().String()] = bindingDataOf(binding)
	}
	return data, nil
}

func catalogDataOf(catalogued domain.CatalogPolicy) (catalogData, error) {
	rule, isRuleWorded := wordFor(policyRuleByWord, catalogued.Rule())
	direction, isDirectionWorded := wordFor(policyDirectionByWord, catalogued.Direction())
	if !isRuleWorded || !isDirectionWorded {
		return catalogData{}, fmt.Errorf("%w: %s", errUnshippedWord, catalogued.Name())
	}
	groups := make([]groupData, 0, len(catalogued.Groups()))
	for _, group := range catalogued.Groups() {
		groups = append(groups, groupData{Name: group.Name().String(), Order: group.Order()})
	}
	return catalogData{Rule: rule, Direction: direction, Groups: groups}, nil
}

func wordFor[Meaning comparable](byWord map[string]Meaning, meaning Meaning) (string, bool) {
	for word, worded := range byWord {
		if worded == meaning {
			return word, true
		}
	}
	return "", false
}

func bindingDataOf(binding domain.Binding) bindingData {
	policies := make([]boundPolicyData, 0, len(binding.Policies()))
	for _, bound := range binding.Policies() {
		policies = append(policies, boundPolicyData{
			Use:         bound.Use().String(),
			ImportRoots: importRootDataOf(bound.ImportRoots()),
			Paths:       groupPathsDataOf(bound.Groups()),
		})
	}
	return bindingData{Policies: policies}
}

func importRootDataOf(roots []domain.ImportRoot) []importRootData {
	served := make([]importRootData, 0, len(roots))
	for _, root := range roots {
		served = append(served, importRootData{Prefix: root.Prefix().String(), Path: root.Directory().String()})
	}
	return served
}

func groupPathsDataOf(groups []domain.GroupPaths) map[string][]string {
	served := make(map[string][]string, len(groups))
	for _, group := range groups {
		globs := make([]string, 0, len(group.Patterns()))
		for _, pattern := range group.Patterns() {
			globs = append(globs, pattern.Glob())
		}
		served[group.Group().String()] = globs
	}
	return served
}
