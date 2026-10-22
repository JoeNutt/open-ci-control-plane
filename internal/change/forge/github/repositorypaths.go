// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const blobEntryType = "blob"

var ErrNoSuchRepository = errors.New("the forge found no such repository")

var ErrTruncatedTree = errors.New("the forge truncated the tree, so not every path is known")

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
	listed, err := exchanged[tree](ctx, paths.forge, defaultBranchTreeReading(repository))
	if err != nil {
		return domain.KnownPaths{}, fmt.Errorf("read the default-branch tree of %s: %w", repository, err)
	}
	if listed.Truncated {
		return domain.KnownPaths{}, fmt.Errorf("%w: %s", ErrTruncatedTree, repository)
	}
	known, err := domain.ParseKnownPaths(listed.blobPaths())
	if err != nil {
		return domain.KnownPaths{}, fmt.Errorf("read the default-branch tree of %s: %w", repository, err)
	}
	return known, nil
}

func defaultBranchTreeReading(repository domain.GovernedRepository) exchange {
	return exchange{
		method:    http.MethodGet,
		endpoint:  fmt.Sprintf("/repos/%s/%s/git/trees/HEAD?recursive=1", repository.Owner(), repository.Name()),
		succeeded: http.StatusOK,
		notFound:  ErrNoSuchRepository,
	}
}

type tree struct {
	Truncated bool        `json:"truncated"`
	Entries   []treeEntry `json:"tree"`
}

type treeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Size int    `json:"size"`
}

func (listed tree) blobPaths() []string {
	var blobs []string
	for _, entry := range listed.Entries {
		if entry.Type == blobEntryType {
			blobs = append(blobs, entry.Path)
		}
	}
	return blobs
}

func (listed tree) blobFiles() []domain.ListedFile {
	var blobs []domain.ListedFile
	for _, entry := range listed.Entries {
		if entry.Type == blobEntryType {
			blobs = append(blobs, domain.ListedFile{Path: entry.Path, Bytes: entry.Size})
		}
	}
	return blobs
}
