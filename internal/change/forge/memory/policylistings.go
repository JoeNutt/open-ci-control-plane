// SPDX-License-Identifier: GPL-3.0-or-later

package memory

import (
	"context"
	"fmt"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type PolicyListings struct {
	Listings map[domain.PolicyCommit][]domain.ListedFile
}

func (listings PolicyListings) ListedFilesAt(
	_ context.Context, at domain.PolicyCommit,
) ([]domain.ListedFile, error) {
	listed, recorded := listings.Listings[at]
	if !recorded {
		return nil, fmt.Errorf("%w: %s at %s", domain.ErrNoSuchCommit, at.Repository(), at.Commit())
	}
	return slices.Clone(listed), nil
}
