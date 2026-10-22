// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var _ domain.PolicyContents = PolicyContents{}

type PolicyContents struct {
	contents FileContents
}

func ParsePolicyContents(settings Settings) (PolicyContents, error) {
	contents, err := ParseFileContents(settings, domain.MaxPolicyFileBytes)
	if err != nil {
		return PolicyContents{}, err
	}
	return PolicyContents{contents: contents}, nil
}

func (policy PolicyContents) ContentsAt(
	ctx context.Context, at domain.PolicyCommit, path domain.RepositoryPath,
) ([]byte, error) {
	read, err := policy.contents.decodedAt(ctx, contentsLocation{repository: at.Repository(), commit: at.Commit(), path: path})
	if err != nil {
		return nil, fmt.Errorf("read %s in %s at %s: %w", path, at.Repository(), at.Commit(), err)
	}
	return read, nil
}
