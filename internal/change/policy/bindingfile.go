// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type bindingDocument struct {
	SchemaVersion *int                 `yaml:"schema_version"`
	Repository    string               `yaml:"repository"`
	Policies      []bindingPolicyEntry `yaml:"policies"`
	Tests         bindingTestsEntry    `yaml:"tests"`
}

type bindingPolicyEntry struct {
	Use         string              `yaml:"use"`
	ImportRoots []importRootEntry   `yaml:"import_roots"`
	Paths       map[string][]string `yaml:"paths"`
}

type importRootEntry struct {
	Prefix string  `yaml:"prefix"`
	Path   *string `yaml:"path"`
}

type bindingTestsEntry struct {
	Workflow  string   `yaml:"workflow"`
	AlwaysRun []string `yaml:"always_run"`
}

func ParseBindingFile(content []byte) (domain.Binding, error) {
	parsed, err := bindingIn(content)
	if err != nil {
		return domain.Binding{}, fmt.Errorf("binding file: %w", err)
	}
	return parsed, nil
}

func bindingIn(content []byte) (domain.Binding, error) {
	document, err := decodedStrictly[bindingDocument](content)
	if err != nil {
		return domain.Binding{}, err
	}
	return domain.ParseBinding(bindingStatementOf(document))
}

func bindingStatementOf(document bindingDocument) domain.BindingStatement {
	return domain.BindingStatement{
		Repository: document.Repository,
		Workflow:   document.Tests.Workflow,
		AlwaysRun:  document.Tests.AlwaysRun,
		Policies:   boundPolicyStatementsOf(document.Policies),
	}
}

func boundPolicyStatementsOf(entries []bindingPolicyEntry) []domain.BoundPolicyStatement {
	policies := make([]domain.BoundPolicyStatement, 0, len(entries))
	for _, entry := range entries {
		policies = append(policies, domain.BoundPolicyStatement{
			Use:         entry.Use,
			ImportRoots: importRootStatementsOf(entry.ImportRoots),
			Paths:       entry.Paths,
		})
	}
	return policies
}

func importRootStatementsOf(entries []importRootEntry) []domain.ImportRootStatement {
	roots := make([]domain.ImportRootStatement, 0, len(entries))
	for _, entry := range entries {
		roots = append(roots, domain.ImportRootStatement{Prefix: entry.Prefix, Path: *entry.Path})
	}
	return roots
}
