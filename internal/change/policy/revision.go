// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"fmt"
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const catalogDirectory = "catalog/"

type parsedFiles struct {
	catalog  []domain.CatalogPolicy
	bindings []domain.Binding
}

func ParseRevision(files []domain.PolicyFile) (domain.PolicyModel, error) {
	bounded, err := domain.ParseBoundedPolicyFiles(files)
	if err != nil {
		return domain.PolicyModel{}, fmt.Errorf("policy revision: %w", err)
	}
	var parsed parsedFiles
	for _, file := range bounded.Files() {
		if err := parsed.add(file); err != nil {
			return domain.PolicyModel{}, fmt.Errorf("policy revision, %s: %w", file.Path, err)
		}
	}
	model, err := domain.ParsePolicyModel(parsed.catalog, parsed.bindings)
	if err != nil {
		return domain.PolicyModel{}, fmt.Errorf("policy revision: %w", err)
	}
	return model, nil
}

func (parsed *parsedFiles) add(file domain.PolicyFile) error {
	if strings.HasPrefix(file.Path, catalogDirectory) {
		policy, err := ParseCatalogFile(file.Content)
		if err != nil {
			return err
		}
		parsed.catalog = append(parsed.catalog, policy)
		return nil
	}
	binding, err := ParseBindingFile(file.Content)
	if err != nil {
		return err
	}
	parsed.bindings = append(parsed.bindings, binding)
	return nil
}
