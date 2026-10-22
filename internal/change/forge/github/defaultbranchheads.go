// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type DefaultBranchHeads struct {
	forge connection
}

func ParseDefaultBranchHeads(settings Settings) (DefaultBranchHeads, error) {
	forge, err := parseCredentialsConnection(settings)
	if err != nil {
		return DefaultBranchHeads{}, err
	}
	return DefaultBranchHeads{forge: forge}, nil
}

func (heads DefaultBranchHeads) DefaultBranchHeadOf(
	ctx context.Context, repository domain.PolicyRepository,
) (domain.CommitSHA, error) {
	branch, err := heads.defaultBranchOf(ctx, repository)
	if err != nil {
		return domain.CommitSHA{}, fmt.Errorf("read the default branch of %s: %w", repository, err)
	}
	head, err := heads.headOf(ctx, repository, branch)
	if err != nil {
		return domain.CommitSHA{}, fmt.Errorf("read the head of %s in %s: %w", branch, repository, err)
	}
	return head, nil
}

type repositoryAnswer struct {
	DefaultBranch string `json:"default_branch"`
}

func (heads DefaultBranchHeads) defaultBranchOf(
	ctx context.Context, repository domain.PolicyRepository,
) (string, error) {
	described, err := jsonAt[repositoryAnswer](ctx, heads.forge, repositoryEndpoint(repository))
	if err != nil {
		return "", err
	}
	return described.DefaultBranch, nil
}

type branchReferenceAnswer struct {
	Object struct {
		SHA string `json:"sha"`
	} `json:"object"`
}

func (heads DefaultBranchHeads) headOf(
	ctx context.Context, repository domain.PolicyRepository, branch string,
) (domain.CommitSHA, error) {
	reference, err := jsonAt[branchReferenceAnswer](ctx, heads.forge, branchReferenceEndpoint(repository, branch))
	if err != nil {
		return domain.CommitSHA{}, err
	}
	return domain.ParseCommitSHA(reference.Object.SHA)
}

func branchReferenceEndpoint(repository domain.PolicyRepository, branch string) string {
	return repositoryEndpoint(repository) + "/git/ref/heads/" + branch
}
