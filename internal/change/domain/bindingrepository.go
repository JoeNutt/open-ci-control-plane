// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

type GovernedRepository struct {
	repositoryName
}

func ParseGovernedRepository(stated string) (GovernedRepository, error) {
	named, err := parseRepositoryName(stated)
	if err != nil {
		return GovernedRepository{}, fmt.Errorf("governed repository: %w", err)
	}
	return GovernedRepository{repositoryName: named}, nil
}

func ParseGovernedRepositorySegments(owner, name string) (GovernedRepository, error) {
	return ParseGovernedRepository(owner + repositorySeparator + name)
}
