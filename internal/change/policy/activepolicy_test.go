// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

const (
	firstPolicyCommit  = "1111111111111111111111111111111111111111"
	secondPolicyCommit = "2222222222222222222222222222222222222222"
)

func TestActivePolicyHoldsNoRevisionBeforeOneIsAccepted(t *testing.T) {
	active := policy.NewActivePolicy()

	if _, err := active.Accepted(); !errors.Is(err, policy.ErrNoPolicyLoaded) {
		t.Fatalf("a fresh active policy gave %v, want ErrNoPolicyLoaded", err)
	}
}

func TestActivePolicyAcceptsAValidCandidateAtItsCommit(t *testing.T) {
	active := activeAtFirstCommit(t)

	if revision := acceptedRevisionOf(t, active); revision != policyCommit(t, firstPolicyCommit) {
		t.Errorf("accepted revision %s, want %s", revision, firstPolicyCommit)
	}
}

func TestActivePolicyKeepsThePreviousRevisionWhenACandidateIsRefused(t *testing.T) {
	active := activeAtFirstCommit(t)
	candidate := liveFiles(t)
	candidate[1].Content = []byte(liveBindingPlacingOnePathInTwoGroups(t))

	if err := active.Offer(policyCommit(t, secondPolicyCommit), candidate); !errors.Is(err, domain.ErrPathInTwoGroups) {
		t.Fatalf("a binding placing one path in two groups gave %v, want ErrPathInTwoGroups", err)
	}
	if revision := acceptedRevisionOf(t, active); revision != policyCommit(t, firstPolicyCommit) {
		t.Errorf("accepted revision %s after a refusal, want the previous %s", revision, firstPolicyCommit)
	}
}

func activeAtFirstCommit(t *testing.T) *policy.ActivePolicy {
	t.Helper()

	active := policy.NewActivePolicy()
	if err := active.Offer(policyCommit(t, firstPolicyCommit), liveFiles(t)); err != nil {
		t.Fatalf("offer the live policy files: %v", err)
	}
	return active
}

func acceptedRevisionOf(t *testing.T, active *policy.ActivePolicy) domain.CommitSHA {
	t.Helper()

	accepted, err := active.Accepted()
	if err != nil {
		t.Fatalf("accepted revision: %v", err)
	}
	return accepted.Revision()
}

func policyCommit(t *testing.T, stated string) domain.CommitSHA {
	t.Helper()

	commit, err := domain.ParseCommitSHA(stated)
	if err != nil {
		t.Fatalf("parse commit %s: %v", stated, err)
	}
	return commit
}
