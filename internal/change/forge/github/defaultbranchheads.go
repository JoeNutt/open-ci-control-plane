// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrNoDefaultBranch = errors.New("the forge named no default branch for the repository")

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
	branch, err := defaultBranchOf(ctx, heads.forge, repository)
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

func defaultBranchOf(ctx context.Context, forge connection, repository ownedRepository) (string, error) {
	described, err := exchanged[repositoryAnswer](ctx, forge,
		policyRepositoryReading(repositoryEndpoint(repository), domain.ErrNoSuchRepository))
	if err != nil {
		return "", err
	}
	if described.DefaultBranch == "" {
		return "", ErrNoDefaultBranch
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
	reference, err := exchanged[branchReferenceAnswer](ctx, heads.forge,
		policyRepositoryReading(branchReferenceEndpoint(repository, branch), domain.ErrNoSuchBranch))
	if err != nil {
		return domain.CommitSHA{}, err
	}
	return domain.ParseCommitSHA(reference.Object.SHA)
}

func branchReferenceEndpoint(repository domain.PolicyRepository, branch string) string {
	return repositoryEndpoint(repository) + "/git/ref/heads/" + url.PathEscape(branch)
}

func policyRepositoryReading(endpoint string, notFound error) exchange {
	return exchange{
		method:    http.MethodGet,
		endpoint:  endpoint,
		succeeded: http.StatusOK,
		notFound:  notFound,
	}
}
