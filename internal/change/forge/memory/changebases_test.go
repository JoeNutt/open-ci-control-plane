// SPDX-License-Identifier: GPL-3.0-or-later

package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

func TestChangeBasesAnswersTheBaseCommitRecordedForAPullRequest(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	base := commitOf(t, "0a1b2c3d4e5f60718293a4b5c6d7e8f901234567")
	var bases domain.ChangeBase = memory.ChangeBases{
		Bases: map[domain.PullRequestLocator]domain.CommitSHA{locator: base},
	}

	answered, err := bases.BaseCommitOf(context.Background(), locator)

	if err != nil {
		t.Fatalf("base commit of %s: %v", locator, err)
	}
	if answered != base {
		t.Fatalf("base commit is %s, want %s", answered, base)
	}
}

func TestChangeBasesRefusesAPullRequestItHoldsNoBaseFor(t *testing.T) {
	_, err := memory.ChangeBases{}.BaseCommitOf(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, domain.ErrNoSuchPullRequest) {
		t.Fatalf("error is %v, want %v", err, domain.ErrNoSuchPullRequest)
	}
}
