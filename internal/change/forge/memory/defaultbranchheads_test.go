// SPDX-License-Identifier: GPL-3.0-or-later

package memory_test

import (
	"context"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

func TestDefaultBranchHeadsAnswersTheHeadRecordedForARepository(t *testing.T) {
	repository := policyRepositoryOf(t, "JoeNutt/open-ci-policy")
	head := commitOf(t, "637d049f40a67db2936d7c97668b07c531b4dcc5")
	heads := memory.DefaultBranchHeads{
		Heads: map[domain.PolicyRepository]domain.CommitSHA{repository: head},
	}

	answered, err := heads.DefaultBranchHeadOf(context.Background(), repository)

	if err != nil {
		t.Fatalf("default branch head of %s: %v", repository, err)
	}
	if answered != head {
		t.Fatalf("default branch head is %s, want %s", answered, head)
	}
}

func policyRepositoryOf(t *testing.T, stated string) domain.PolicyRepository {
	t.Helper()

	repository, err := domain.ParsePolicyRepository(stated)
	if err != nil {
		t.Fatalf("parse policy repository %q: %v", stated, err)
	}
	return repository
}
