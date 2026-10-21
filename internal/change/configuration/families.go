// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"errors"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func familiesIn(stated []string) ([]domain.PolicyFamily, error) {
	if len(stated) == 0 {
		return nil, families.missing()
	}
	named := make([]domain.PolicyFamily, 0, len(stated))
	for _, name := range stated {
		family, err := domain.ParsePolicyFamily(name)
		if err != nil {
			return nil, families.invalid(err)
		}
		named = append(named, family)
	}
	if len(slices.Compact(slices.Sorted(slices.Values(named)))) != len(named) {
		return nil, families.invalid(errors.New("repeated"))
	}
	return named, nil
}
