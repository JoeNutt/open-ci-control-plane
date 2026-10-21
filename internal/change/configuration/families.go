// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"

func familiesIn(stated []string) ([]domain.PolicyFamily, error) {
	if len(stated) == 0 {
		return nil, families.missing()
	}
	named := make([]domain.PolicyFamily, 0, len(stated))
	for _, name := range stated {
		named = append(named, domain.PolicyFamily(name))
	}
	return named, nil
}
