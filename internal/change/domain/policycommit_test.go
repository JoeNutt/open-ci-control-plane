// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const policyHead = "637d049f40a67db2936d7c97668b07c531b4dcc5"

func TestParsePolicyCommitKeepsItsRepositoryAndCommit(t *testing.T) {
	repository, commit := parsedPolicyRepository(t), parsedCommit(t, policyHead)

	at, err := domain.ParsePolicyCommit(repository, commit)

	if err != nil {
		t.Fatalf("parse policy commit: %v", err)
	}
	if at.Repository() != repository || at.Commit() != commit {
		t.Errorf("policy commit is %s at %s, want %s at %s", at.Repository(), at.Commit(), repository, commit)
	}
}

func TestParsePolicyCommitRefusesAnUnparsedRepositoryOrCommit(t *testing.T) {
	if _, err := domain.ParsePolicyCommit(parsedPolicyRepository(t), domain.CommitSHA{}); !errors.Is(err, domain.ErrMalformedCommitSHA) {
		t.Errorf("a zero commit gave %v, want ErrMalformedCommitSHA", err)
	}
	if _, err := domain.ParsePolicyCommit(domain.PolicyRepository{}, parsedCommit(t, policyHead)); !errors.Is(err, domain.ErrMalformedRepository) {
		t.Errorf("a zero repository gave %v, want ErrMalformedRepository", err)
	}
}

func parsedPolicyRepository(t *testing.T) domain.PolicyRepository {
	t.Helper()

	repository, err := domain.ParsePolicyRepository(policyRepository)
	if err != nil {
		t.Fatalf("parse policy repository: %v", err)
	}
	return repository
}

func parsedCommit(t *testing.T, stated string) domain.CommitSHA {
	t.Helper()

	commit, err := domain.ParseCommitSHA(stated)
	if err != nil {
		t.Fatalf("parse commit %s: %v", stated, err)
	}
	return commit
}
