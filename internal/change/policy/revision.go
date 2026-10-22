// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	catalogDirectory  = "catalog/"
	bindingsDirectory = "bindings/"
)

var ErrUnplacedPolicyFile = errors.New("policy file is neither in catalog/ nor in bindings/")

type parsedFiles struct {
	catalog  []domain.CatalogPolicy
	bindings []domain.Binding
}

type placement struct {
	directory string
	add       func(*parsedFiles, []byte) error
}

var placementsOfPolicyFiles = []placement{
	{directory: catalogDirectory, add: (*parsedFiles).addCatalogPolicy},
	{directory: bindingsDirectory, add: (*parsedFiles).addBinding},
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
	placed := placementIndexOf(file.Path)
	if placed < 0 {
		return ErrUnplacedPolicyFile
	}
	return placementsOfPolicyFiles[placed].add(parsed, file.Content)
}

func (parsed *parsedFiles) addCatalogPolicy(content []byte) error {
	policy, err := ParseCatalogFile(content)
	if err != nil {
		return err
	}
	parsed.catalog = append(parsed.catalog, policy)
	return nil
}

func (parsed *parsedFiles) addBinding(content []byte) error {
	binding, err := ParseBindingFile(content)
	if err != nil {
		return err
	}
	parsed.bindings = append(parsed.bindings, binding)
	return nil
}

func isPlacedPolicyFile(path string) bool { return placementIndexOf(path) >= 0 }

func placementIndexOf(path string) int {
	return slices.IndexFunc(placementsOfPolicyFiles, func(candidate placement) bool {
		return strings.HasPrefix(path, candidate.directory)
	})
}
