// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrNoSuchCommit = errors.New("the forge reports no such commit in the repository")

type PolicyCommit struct {
	repository PolicyRepository
	commit     CommitSHA
}

func ParsePolicyCommit(repository PolicyRepository, commit CommitSHA) (PolicyCommit, error) {
	if repository == (PolicyRepository{}) {
		return PolicyCommit{}, fmt.Errorf("policy commit: %w: no repository", ErrMalformedRepository)
	}
	if commit == (CommitSHA{}) {
		return PolicyCommit{}, fmt.Errorf("policy commit: %w: no commit", ErrMalformedCommitSHA)
	}
	return PolicyCommit{repository: repository, commit: commit}, nil
}

func (at PolicyCommit) Repository() PolicyRepository { return at.repository }

func (at PolicyCommit) Commit() CommitSHA { return at.commit }
