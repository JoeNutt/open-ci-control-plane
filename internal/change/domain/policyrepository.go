// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

type PolicyRepository struct {
	repository GovernedRepository
}

func ParsePolicyRepository(stated string) (PolicyRepository, error) {
	repository, err := ParseGovernedRepository(stated)
	if err != nil {
		return PolicyRepository{}, fmt.Errorf("policy repository: %w", err)
	}
	return PolicyRepository{repository: repository}, nil
}

func (policy PolicyRepository) String() string { return policy.repository.String() }

func (policy PolicyRepository) Owner() string { return policy.repository.Owner() }

func (policy PolicyRepository) Name() string { return policy.repository.Name() }
