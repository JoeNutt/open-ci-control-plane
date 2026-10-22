// SPDX-License-Identifier: GPL-3.0-or-later

package memory

import (
	"context"
	"fmt"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type PolicyFileAt struct {
	At   domain.PolicyCommit
	Path domain.RepositoryPath
}

type PolicyContents struct {
	Files map[PolicyFileAt][]byte
}

func (contents PolicyContents) ContentsAt(
	_ context.Context, at domain.PolicyCommit, path domain.RepositoryPath,
) ([]byte, error) {
	recorded, isRecorded := contents.Files[PolicyFileAt{At: at, Path: path}]
	if !isRecorded {
		return nil, fmt.Errorf("%w: %s in %s at %s", domain.ErrNoSuchFile, path, at.Repository(), at.Commit())
	}
	return slices.Clone(recorded), nil
}
