// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"errors"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var errRepeatedFamily = errors.New("a policy family is named twice")

func familiesIn(stated statedItems) ([]domain.PolicyFamily, error) {
	names, err := valueOf[[]string](stated, familiesItem)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, familiesItem.invalid(errEmptyList)
	}
	families := make([]domain.PolicyFamily, 0, len(names))
	for _, name := range names {
		family, err := domain.ParsePolicyFamily(name)
		if err != nil {
			return nil, familiesItem.invalid(err)
		}
		families = append(families, family)
	}
	if repeatsAnEntry(families) {
		return nil, familiesItem.invalid(errRepeatedFamily)
	}
	return families, nil
}
