// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	baseSHA      = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4"
	headTipSHA   = "9f1c8d2e4b7a60718293a4b5c6d7e8f901234567"
	mergeBaseSHA = "0a1b2c3d4e5f60718293a4b5c6d7e8f9aabbccdd"
)

func TestBaseCommitOfIsTheMergeBaseEvenAfterTheBaseBranchMoved(t *testing.T) {
	server := servingEach(t, map[string]string{
		"/repos/acme/payments/pulls/412": `{"number": 412,
			"base": {"ref": "main", "sha": "` + baseSHA + `"}, "head": {"sha": "` + headTipSHA + `"}}`,
		"/repos/acme/payments/compare/" + baseSHA + "..." + headTipSHA: `{
			"merge_base_commit": {"sha": "` + mergeBaseSHA + `"}}`,
	})

	base, err := sourceAt(t, server).BaseCommitOf(context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("base commit of: %v", err)
	}
	if base.String() != mergeBaseSHA {
		t.Errorf("base commit is %q, want the merge base %q, not the base branch tip %q",
			base, mergeBaseSHA, baseSHA)
	}
}

func TestBaseCommitOfRefusesABaseTheForgeDoesNotStateAsAFullCommitSHA(t *testing.T) {
	answers := map[string]string{
		"no base":    `{"number": 412}`,
		"no sha":     `{"number": 412, "base": {"ref": "main"}}`,
		"short sha":  `{"number": 412, "base": {"sha": "e3b0c44"}}`,
		"upper case": `{"number": 412, "base": {"sha": "` + strings.ToUpper(baseSHA) + `"}}`,
		"a ref":      `{"number": 412, "base": {"sha": "main"}}`,
	}

	for name, answer := range answers {
		server := answering(t, http.StatusOK, answer)

		_, err := sourceAt(t, server).BaseCommitOf(context.Background(), locatorOf(t, "acme/payments", 412))

		if !errors.Is(err, domain.ErrMalformedCommitSHA) {
			t.Errorf("%s: error is %v, want %v", name, err, domain.ErrMalformedCommitSHA)
		}
	}
}

func TestBaseCommitOfReportsAPullRequestTheForgeDoesNotKnowWithoutTheCredential(t *testing.T) {
	server := answering(t, http.StatusNotFound, `{"message": "Not Found `+forgeToken+`"}`)

	_, err := sourceAt(t, server).BaseCommitOf(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, domain.ErrNoSuchPullRequest) {
		t.Fatalf("error is %v, want %v", err, domain.ErrNoSuchPullRequest)
	}
	if !strings.Contains(err.Error(), "acme/payments#412") {
		t.Errorf("error %q does not name the pull request", err)
	}
	mustRevealNoneOf(t, "base commit refusal", err.Error(), []string{forgeToken})
}
