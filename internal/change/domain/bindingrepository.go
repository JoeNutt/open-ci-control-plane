// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type GovernedRepository struct {
	repositoryName
}

func ParseGovernedRepository(stated string) (GovernedRepository, error) {
	named, err := parseRepositoryName(stated)
	if err != nil {
		return GovernedRepository{}, err
	}
	return GovernedRepository{repositoryName: named}, nil
}
