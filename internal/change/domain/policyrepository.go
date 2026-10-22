// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

type PolicyRepository struct {
	repositoryName
}

func ParsePolicyRepository(stated string) (PolicyRepository, error) {
	named, err := parseRepositoryName(stated)
	if err != nil {
		return PolicyRepository{}, fmt.Errorf("policy repository: %w", err)
	}
	return PolicyRepository{repositoryName: named}, nil
}
