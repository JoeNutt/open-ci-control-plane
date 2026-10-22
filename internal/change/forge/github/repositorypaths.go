// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type RepositoryPaths struct {
	forge connection
}

func ParseRepositoryPaths(settings Settings) (RepositoryPaths, error) {
	forge, err := parseCredentialsConnection(settings)
	if err != nil {
		return RepositoryPaths{}, err
	}
	return RepositoryPaths{forge: forge}, nil
}

type tree struct{}

func (paths RepositoryPaths) KnownPathsOf(ctx context.Context, repository domain.GovernedRepository) (domain.KnownPaths, error) {
	_, err := jsonAt[tree](ctx, paths.forge, defaultBranchTreeEndpoint(repository))
	return domain.KnownPaths{}, err
}

func defaultBranchTreeEndpoint(repository domain.GovernedRepository) string {
	return fmt.Sprintf("/repos/%s/%s/git/trees/HEAD?recursive=1", repository.Owner(), repository.Name())
}
