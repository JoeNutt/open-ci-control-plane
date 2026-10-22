// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

const (
	governedRepository = "acme/payments"
	pullRequestNumber  = 412
	baseSHA            = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4"
	headTipSHA         = "9f1c8d2e4b7a60718293a4b5c6d7e8f901234567"
	mergeBaseSHA       = "0a1b2c3d4e5f60718293a4b5c6d7e8f9aabbccdd"
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
		context.Background(), locatorOf(t, governedRepository, pullRequestNumber), tipsOf(t))

	if err != nil {
		t.Fatalf("merge base of: %v", err)
	}
	if mergeBase.String() != mergeBaseSHA {
		t.Errorf("merge base is %q, want %q, not the base branch tip %q", mergeBase, mergeBaseSHA, baseSHA)
	}
	want := recordedRequest{
		method:        http.MethodGet,
		escapedPath:   "/repos/" + governedRepository + "/compare/" + baseSHA + "..." + headTipSHA,
		authorization: "Bearer " + forgeToken,
	}
	if asked != want {
		t.Errorf("the adapter asked %+v, want %+v", asked, want)
	}
}

func TestMergeBaseOfRefusesAComparisonWithoutAFullMergeBase(t *testing.T) {
	answers := map[string]string{
		"no merge base": `{"status": "diverged"}`,
		"no sha":        `{"merge_base_commit": {}}`,
		"short sha":     `{"merge_base_commit": {"sha": "0a1b2c3"}}`,
		"upper case":    `{"merge_base_commit": {"sha": "` + strings.ToUpper(mergeBaseSHA) + `"}}`,
	}

	for name, answer := range answers {
		server := answering(t, http.StatusOK, answer)

		_, err := sourceAt(t, server).MergeBaseOf(
			context.Background(), locatorOf(t, governedRepository, pullRequestNumber), tipsOf(t))

		if !errors.Is(err, domain.ErrMalformedCommitSHA) {
			t.Errorf("%s: error is %v, want %v", name, err, domain.ErrMalformedCommitSHA)
		}
	}
}

func TestMergeBaseOfReportsTipsTheForgeCannotCompareWithoutTheCredential(t *testing.T) {
	server := answering(t, http.StatusNotFound, `{"message": "Not Found `+forgeToken+`"}`)

	_, err := sourceAt(t, server).MergeBaseOf(
		context.Background(), locatorOf(t, governedRepository, pullRequestNumber), tipsOf(t))

	if !errors.Is(err, github.ErrTipsNotComparable) {
		t.Fatalf("error is %v, want %v", err, github.ErrTipsNotComparable)
	}
	if !strings.Contains(err.Error(), governedRepository) {
		t.Errorf("error %q does not name the repository", err)
	}
	mustRevealNoneOf(t, "merge base refusal", err.Error(), []string{forgeToken})
}
