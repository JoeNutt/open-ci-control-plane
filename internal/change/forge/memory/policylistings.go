// SPDX-License-Identifier: GPL-3.0-or-later

package memory

import (
	"context"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type PolicyListings struct {
	Listings map[domain.PolicyCommit][]domain.ListedFile
}

func (listings PolicyListings) ListedFilesAt(
	_ context.Context, at domain.PolicyCommit,
) ([]domain.ListedFile, error) {
	return listings.Listings[at], nil
}
