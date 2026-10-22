// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"context"
	"fmt"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type recordedHeads struct {
	Heads map[domain.PolicyRepository]domain.CommitSHA
}

func (heads recordedHeads) DefaultBranchHeadOf(
	_ context.Context, repository domain.PolicyRepository,
) (domain.CommitSHA, error) {
	head, isRecorded := heads.Heads[repository]
	if !isRecorded {
		return domain.CommitSHA{}, fmt.Errorf("%w: %s", domain.ErrNoSuchRepository, repository)
	}
	return head, nil
}

type recordedListings struct {
	Listings map[domain.PolicyCommit][]domain.ListedFile
}

func (listings recordedListings) ListedFilesAt(_ context.Context, at domain.PolicyCommit) ([]domain.ListedFile, error) {
	listed, isRecorded := listings.Listings[at]
	if !isRecorded {
		return nil, fmt.Errorf("%w: %s at %s", domain.ErrNoSuchCommit, at.Repository(), at.Commit())
	}
	return slices.Clone(listed), nil
}

type policyFileAt struct {
	At   domain.PolicyCommit
	Path domain.RepositoryPath
}

type recordedPolicyFiles struct {
	Files map[policyFileAt][]byte
}

func (contents recordedPolicyFiles) ContentsAt(
	_ context.Context, at domain.PolicyCommit, path domain.RepositoryPath,
) ([]byte, error) {
	recorded, isRecorded := contents.Files[policyFileAt{At: at, Path: path}]
	if !isRecorded {
		return nil, fmt.Errorf("%w: %s in %s at %s", domain.ErrNoSuchFile, path, at.Repository(), at.Commit())
	}
	return slices.Clone(recorded), nil
}
