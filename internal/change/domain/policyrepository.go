// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrNoSuchRepository = errors.New("the forge reports no such repository")

type PolicyRepository struct {
	named repositoryName
}

func ParsePolicyRepository(stated string) (PolicyRepository, error) {
	named, err := parseRepositoryName(stated)
	if err != nil {
		return PolicyRepository{}, fmt.Errorf("policy repository: %w", err)
	}
	return PolicyRepository{named: named}, nil
}

func (repository PolicyRepository) String() string { return repository.named.String() }

func (repository PolicyRepository) Owner() string { return repository.named.Owner() }

func (repository PolicyRepository) Name() string { return repository.named.Name() }
