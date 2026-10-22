// SPDX-License-Identifier: GPL-3.0-or-later

package memory_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

func TestPolicyListingsAnswersTheFilesRecordedAtACommit(t *testing.T) {
	at := recordedPolicyCommit(t)
	recorded := []domain.ListedFile{{Path: "catalog/clean-architecture.yaml", Bytes: 242}}
	listings := memory.PolicyListings{Listings: map[domain.PolicyCommit][]domain.ListedFile{at: recorded}}

	answered, err := listings.ListedFilesAt(context.Background(), at)

	if err != nil {
		t.Fatalf("listed files at %+v: %v", at, err)
	}
	if !slices.Equal(answered, recorded) {
		t.Fatalf("listed files are %+v, want %+v", answered, recorded)
	}
}

func TestPolicyListingsRefusesACommitItHoldsNoListingFor(t *testing.T) {
	at := recordedPolicyCommit(t)

	_, err := memory.PolicyListings{}.ListedFilesAt(context.Background(), at)

	if !errors.Is(err, domain.ErrNoSuchCommit) {
		t.Fatalf("error is %v, want %v", err, domain.ErrNoSuchCommit)
	}
}

func recordedPolicyCommit(t *testing.T) domain.PolicyCommit {
	t.Helper()

	return domain.PolicyCommit{
		Repository: policyRepositoryOf(t, recordedPolicyRepository),
		Commit:     commitOf(t, "637d049f40a67db2936d7c97668b07c531b4dcc5"),
	}
}
