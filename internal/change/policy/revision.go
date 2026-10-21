// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func ParseRevision(files []domain.PolicyFile) (domain.PolicyModel, error) {
	bounded, err := domain.ParseBoundedPolicyFiles(files)
	if err != nil {
		return domain.PolicyModel{}, err
	}
	var catalog []domain.CatalogPolicy
	var bindings []domain.Binding
	for _, file := range bounded.Files() {
		if strings.HasPrefix(file.Path, "catalog/") {
			parsed, err := ParseCatalogFile(file.Content)
			if err != nil {
				return domain.PolicyModel{}, err
			}
			catalog = append(catalog, parsed)
		} else {
			parsed, err := ParseBindingFile(file.Content)
			if err != nil {
				return domain.PolicyModel{}, err
			}
			bindings = append(bindings, parsed)
		}
	}
	return domain.ParsePolicyModel(catalog, bindings)
}
