// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"strings"
)

type GovernedRepository struct {
	owner string
	name  string
}

func ParseGovernedRepository(stated string) (GovernedRepository, error) {
	if len(stated) > maxRepositoryLength {
		return GovernedRepository{}, fmt.Errorf("%w: %d bytes exceeds %d",
			ErrMalformedRepository, len(stated), maxRepositoryLength)
	}
	owner, name, separated := strings.Cut(stated, repositorySeparator)
	if !separated || !ownerRule.admits(owner) || !nameRule.admits(name) {
		return GovernedRepository{}, fmt.Errorf("%w: %d bytes", ErrMalformedRepository, len(stated))
	}
	return GovernedRepository{owner: owner, name: name}, nil
}

func ParseGovernedRepositorySegments(owner, name string) (GovernedRepository, error) {
	return ParseGovernedRepository(owner + repositorySeparator + name)
}

func (repository GovernedRepository) String() string {
	return repository.owner + repositorySeparator + repository.name
}

func (repository GovernedRepository) Owner() string { return repository.owner }

func (repository GovernedRepository) Name() string { return repository.name }
