// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	baseSHA      = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4"
	headTipSHA   = "9f1c8d2e4b7a60718293a4b5c6d7e8f901234567"
	mergeBaseSHA = "0a1b2c3d4e5f60718293a4b5c6d7e8f9aabbccdd"
)

func tipsOf(t *testing.T) domain.BranchTips {
	t.Helper()

	return domain.BranchTips{Base: commitOf(t, baseSHA), Head: commitOf(t, headTipSHA)}
}

func commitOf(t *testing.T, stated string) domain.CommitSHA {
	t.Helper()

	commit, err := domain.ParseCommitSHA(stated)
	if err != nil {
		t.Fatalf("parse commit %q: %v", stated, err)
	}
	return commit
}

func TestMergeBaseOfAsksOnlyForTheComparisonOfTheTipsAlreadyRead(t *testing.T) {
	var asked recordedRequest
	server := recording(t, &asked, `{"merge_base_commit": {"sha": "`+mergeBaseSHA+`"}}`)

	mergeBase, err := sourceAt(t, server).MergeBaseOf(
		context.Background(), locatorOf(t, "acme/payments", 412), tipsOf(t))

	if err != nil {
		t.Fatalf("merge base of: %v", err)
	}
	if mergeBase.String() != mergeBaseSHA {
		t.Errorf("merge base is %q, want %q, not the base branch tip %q", mergeBase, mergeBaseSHA, baseSHA)
	}
	want := recordedRequest{
		method:        http.MethodGet,
		escapedPath:   "/repos/acme/payments/compare/" + baseSHA + "..." + headTipSHA,
		authorization: "Bearer " + forgeToken,
	}
	if asked != want {
		t.Errorf("the adapter asked %+v, want %+v", asked, want)
	}
}
