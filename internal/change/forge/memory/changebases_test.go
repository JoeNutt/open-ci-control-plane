// SPDX-License-Identifier: GPL-3.0-or-later

package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

func recordedTips(t *testing.T) domain.BranchTips {
	t.Helper()

	tips, err := domain.ParseBranchTips(
		commitOf(t, "7c6b5a4938271605f4e3d2c1b0a9f8e7d6c5b4a3"),
		commitOf(t, "9f1cabc0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6"))
	if err != nil {
		t.Fatalf("parse branch tips: %v", err)
	}
	return tips
}

func TestChangeBasesAnswersTheMergeBaseRecordedForTheTips(t *testing.T) {
	tips := recordedTips(t)
	mergeBase := commitOf(t, "0a1b2c3d4e5f60718293a4b5c6d7e8f901234567")
	var bases domain.ChangeBase = memory.ChangeBases{
		MergeBases: map[domain.BranchTips]domain.CommitSHA{tips: mergeBase},
	}

	answered, err := bases.MergeBaseOf(context.Background(), locatorOf(t, "acme/payments", 412), tips)

	if err != nil {
		t.Fatalf("merge base of: %v", err)
	}
	if answered != mergeBase {
		t.Fatalf("merge base is %s, want %s", answered, mergeBase)
	}
}

func TestChangeBasesRefusesTipsItHoldsNoMergeBaseFor(t *testing.T) {
	_, err := memory.ChangeBases{}.MergeBaseOf(
		context.Background(), locatorOf(t, "acme/payments", 412), recordedTips(t))

	if !errors.Is(err, domain.ErrNoMergeBase) {
		t.Fatalf("error is %v, want %v", err, domain.ErrNoMergeBase)
	}
}
