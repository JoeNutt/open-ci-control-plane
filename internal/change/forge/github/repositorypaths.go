// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const fileEntryType = "blob"

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

func (paths RepositoryPaths) KnownPathsOf(
	ctx context.Context, repository domain.GovernedRepository,
) (domain.KnownPaths, error) {
	listed, err := jsonAt[tree](ctx, paths.forge, defaultBranchTreeEndpoint(repository))
	if err != nil {
		return domain.KnownPaths{}, fmt.Errorf("read the default-branch tree of %s: %w", repository, err)
	}
	if listed.Truncated {
		return domain.KnownPaths{}, ErrTruncatedTree
	}
	return domain.NewKnownPaths(listed.repositoryPaths()), nil
}

func defaultBranchTreeEndpoint(repository domain.GovernedRepository) string {
	return fmt.Sprintf("/repos/%s/%s/git/trees/HEAD?recursive=1", repository.Owner(), repository.Name())
}

var ErrTruncatedTree = errors.New("truncated")

type tree struct {
	Truncated bool        `json:"truncated"`
	Entries   []treeEntry `json:"tree"`
}

type treeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

func (listed tree) repositoryPaths() []domain.RepositoryPath {
	var files []domain.RepositoryPath
	for _, entry := range listed.Entries {
		if path, err := domain.ParseRepositoryPath(entry.Path); entry.Type == fileEntryType && err == nil {
			files = append(files, path)
		}
	}
	return files
}
