// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"strings"
)

type Repository struct {
	owner string
	name  string
}

func ParseRepository(repository string) (Repository, error) {
	if len(repository) > maxRepositoryLength {
		return Repository{}, fmt.Errorf("%w: %d bytes exceeds %d",
			ErrMalformedRepository, len(repository), maxRepositoryLength)
	}
	owner, name, separated := strings.Cut(repository, repositorySeparator)
	if !separated || !ownerRule.admits(owner) || !nameRule.admits(name) {
		return Repository{}, fmt.Errorf("%w: %q", ErrMalformedRepository, repository)
	}
	return Repository{owner: owner, name: name}, nil
}
