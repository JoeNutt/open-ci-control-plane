// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

const firstPolicyCommit = "1111111111111111111111111111111111111111"

func TestActivePolicyHoldsNoRevisionBeforeOneIsAccepted(t *testing.T) {
	active := policy.NewActivePolicy()

	if _, err := active.Accepted(); !errors.Is(err, policy.ErrNoPolicyLoaded) {
		t.Fatalf("a fresh active policy gave %v, want ErrNoPolicyLoaded", err)
	}
}

func TestActivePolicyAcceptsAValidCandidateAtItsCommit(t *testing.T) {
	active := policy.NewActivePolicy()
	revision := policyCommit(t, firstPolicyCommit)

	if err := active.Offer(revision, liveFiles(t)); err != nil {
		t.Fatalf("offer the live policy files: %v", err)
	}
	accepted, err := active.Accepted()
	if err != nil {
		t.Fatalf("accepted revision: %v", err)
	}
	if accepted.Revision() != revision {
		t.Errorf("accepted revision %s, want %s", accepted.Revision(), revision)
	}
}

func policyCommit(t *testing.T, stated string) domain.CommitSHA {
	t.Helper()

	commit, err := domain.ParseCommitSHA(stated)
	if err != nil {
		t.Fatalf("parse commit %s: %v", stated, err)
	}
	return commit
}
