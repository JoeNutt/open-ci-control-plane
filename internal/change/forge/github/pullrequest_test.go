// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

func TestPullRequestAtReportsEveryApprovalTheForgeHolds(t *testing.T) {
	server := servingEach(t, map[string]string{
		"/repos/acme/payments/pulls/412": `{
			"number": 412,
			"user": {"login": "jo", "type": "User"},
			"author_association": "MEMBER"
		}`,
		"/repos/acme/payments/pulls/412/reviews": `[
			{"user": {"login": "sam", "type": "User"}, "state": "APPROVED"},
			{"user": {"login": "kim", "type": "User"}, "state": "COMMENTED"}
		]`,
	})
	source := sourceAt(t, server)

	pullRequest, err := source.PullRequestAt(
		context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("pull request at: %v", err)
	}
	want := domain.Declared[domain.Approval]{{Login: "sam"}}
	if !reflect.DeepEqual(pullRequest.Approvals, want) {
		t.Fatalf("approvals are %+v, want %+v", pullRequest.Approvals, want)
	}
}

func TestPullRequestAtDropsAnApprovalItsReviewerWithdrew(t *testing.T) {
	server := servingEach(t, map[string]string{
		"/repos/acme/payments/pulls/412": `{
			"number": 412,
			"user": {"login": "jo", "type": "User"},
			"author_association": "MEMBER"
		}`,
		"/repos/acme/payments/pulls/412/reviews": `[
			{"user": {"login": "sam", "type": "User"}, "state": "APPROVED"},
			{"user": {"login": "sam", "type": "User"}, "state": "CHANGES_REQUESTED"},
			{"user": {"login": "kim", "type": "User"}, "state": "APPROVED"},
			{"user": {"login": "kim", "type": "User"}, "state": "COMMENTED"}
		]`,
	})
	source := sourceAt(t, server)

	pullRequest, err := source.PullRequestAt(
		context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("pull request at: %v", err)
	}
	want := domain.Declared[domain.Approval]{{Login: "kim"}}
	if !reflect.DeepEqual(pullRequest.Approvals, want) {
		t.Fatalf("approvals are %+v, want %+v", pullRequest.Approvals, want)
	}
}

func TestPullRequestAtReportsTheContextTheForgeStates(t *testing.T) {
	server := servingEach(t, map[string]string{
		"/repos/acme/payments/pulls/412": `{
			"number": 412,
			"draft": false,
			"base": {"ref": "main"},
			"head": {"sha": "9f1c8d2e4b7a"},
			"user": {"login": "jo", "type": "User"},
			"author_association": "MEMBER",
			"labels": [{"name": "backend"}, {"name": "needs-review"}]
		}`,
		"/repos/acme/payments/pulls/412/reviews": `[]`,
	})
	source := sourceAt(t, server)

	pullRequest, err := source.PullRequestAt(
		context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("pull request at: %v", err)
	}
	want := domain.PullRequest{
		Number:     412,
		Repository: "acme/payments",
		BaseRef:    "main",
		HeadSHA:    "9f1c8d2e4b7a",
		Author: domain.Author{
			Login:       "jo",
			Association: domain.AuthorAssociationMember,
		},
		Labels:    domain.Declared[string]{"backend", "needs-review"},
		Approvals: domain.Declared[domain.Approval]{},
	}
	if !reflect.DeepEqual(pullRequest, want) {
		t.Fatalf("pull request is %+v, want %+v", pullRequest, want)
	}
}

func TestPullRequestAtRefusesAnAccountTypeItCannotClassify(t *testing.T) {
	server := serving(t, "/repos/acme/payments/pulls/412", `{
		"number": 412,
		"user": {"login": "jo", "type": "Golem"}
	}`)
	source := sourceAt(t, server)

	_, err := source.PullRequestAt(
		context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, github.ErrUnknownUserType) {
		t.Fatalf("error is %v, want %v", err, github.ErrUnknownUserType)
	}
}
