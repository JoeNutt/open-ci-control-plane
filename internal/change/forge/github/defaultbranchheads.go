// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"

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

type repositoryAnswer struct {
	DefaultBranch string `json:"default_branch"`
}

type referenceAnswer struct {
	Object struct {
		SHA string `json:"sha"`
	} `json:"object"`
}

func (heads DefaultBranchHeads) DefaultBranchHeadOf(
	ctx context.Context, repository domain.PolicyRepository,
) (domain.CommitSHA, error) {
	described, err := jsonAt[repositoryAnswer](ctx, heads.forge, "/repos/"+repository.Owner()+"/"+repository.Name())
	if err != nil {
		return domain.CommitSHA{}, err
	}
	reference, err := jsonAt[referenceAnswer](ctx, heads.forge,
		"/repos/"+repository.Owner()+"/"+repository.Name()+"/git/ref/heads/"+described.DefaultBranch)
	if err != nil {
		return domain.CommitSHA{}, err
	}
	return domain.ParseCommitSHA(reference.Object.SHA)
}
