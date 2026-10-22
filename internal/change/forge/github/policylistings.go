// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var _ domain.PolicyListings = PolicyListings{}

type PolicyListings struct {
	forge connection
}

func ParsePolicyListings(settings Settings) (PolicyListings, error) {
	forge, err := parseCredentialsConnection(settings)
	if err != nil {
		return PolicyListings{}, err
	}
	return PolicyListings{forge: forge}, nil
}

func (listings PolicyListings) ListedFilesAt(
	ctx context.Context, at domain.PolicyCommit,
) ([]domain.ListedFile, error) {
	listed, err := exchanged[tree](ctx, listings.forge,
		policyRepositoryReading(commitTreeEndpoint(at), domain.ErrNoSuchCommit))
	if err != nil {
		return nil, fmt.Errorf("list the files of %s at %s: %w", at.Repository(), at.Commit(), err)
	}
	if listed.Truncated {
		return nil, fmt.Errorf("%w: %s at %s", ErrTruncatedTree, at.Repository(), at.Commit())
	}
	return listed.blobFiles(), nil
}

func commitTreeEndpoint(at domain.PolicyCommit) string {
	return repositoryEndpoint(at.Repository()) + "/git/trees/" + at.Commit().String() + "?recursive=1"
}
