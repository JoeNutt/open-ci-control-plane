// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
)

var ErrNoSuchRepository = errors.New("the forge reports no such repository")

type repositoryName struct {
	owner string
	name  string
}

func parseRepositoryName(stated string) (repositoryName, error) {
	if len(stated) > maxRepositoryLength {
		return repositoryName{}, fmt.Errorf("%w: %d bytes exceeds %d",
			ErrMalformedRepository, len(stated), maxRepositoryLength)
	}
	owner, name, separated := strings.Cut(stated, repositorySeparator)
	if !separated || !ownerRule.admits(owner) || !nameRule.admits(name) {
		return repositoryName{}, fmt.Errorf("%w: %d bytes", ErrMalformedRepository, len(stated))
	}
	return repositoryName{owner: owner, name: name}, nil
}

func (repository repositoryName) String() string {
	return repository.owner + repositorySeparator + repository.name
}

func (repository repositoryName) Owner() string { return repository.owner }

func (repository repositoryName) Name() string { return repository.name }
