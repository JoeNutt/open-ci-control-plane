// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

const policyTreePath = policyRepositoryPath + "/git/trees/" + policyHead

const policyTreeAnswer = `{"sha": "0653ac93140df61ae2a2e2bff3f067d5117ffdfa", "truncated": false, "tree": [
	{"path": "README.md", "mode": "100644", "type": "blob", "size": 375},
	{"path": "bindings", "mode": "040000", "type": "tree"},
	{"path": "bindings/JoeNutt", "mode": "040000", "type": "tree"},
	{"path": "bindings/JoeNutt/open-ci-control-plane.yaml", "mode": "100644", "type": "blob", "size": 820},
	{"path": "catalog", "mode": "040000", "type": "tree"},
	{"path": "catalog/clean-architecture.yaml", "mode": "100644", "type": "blob", "size": 242}
]}`

func TestListedFilesAtListsEveryFileInTheTreeOfTheCommitAsTheInstallation(t *testing.T) {
	var asked []askedRequest
	server := recordingEach(t, &asked, map[string]string{policyTreePath: policyTreeAnswer})

	listed, err := listingsAt(t, server).ListedFilesAt(context.Background(), policyHeadCommit(t))

	if err != nil {
		t.Fatalf("listed files at: %v", err)
	}
	want := []domain.ListedFile{
		{Path: "README.md", Bytes: 375},
		{Path: "bindings/JoeNutt/open-ci-control-plane.yaml", Bytes: 820},
		{Path: "catalog/clean-architecture.yaml", Bytes: 242},
	}
	if !slices.Equal(listed, want) {
		t.Errorf("listed %+v, want %+v", listed, want)
	}
	wantAsked := []askedRequest{{
		method: http.MethodGet, escapedPath: policyTreePath, rawQuery: "recursive=1", authorization: "Bearer " + forgeToken,
	}}
	if !slices.Equal(asked, wantAsked) {
		t.Errorf("the adapter asked %+v, want %+v", asked, wantAsked)
	}
}

func TestListedFilesAtRefusesATreeTheForgeTruncated(t *testing.T) {
	server := serving(t, policyTreePath, `{"truncated": true, "tree": [
		{"path": "catalog/clean-architecture.yaml", "type": "blob", "size": 242}
	]}`)

	_, err := listingsAt(t, server).ListedFilesAt(context.Background(), policyHeadCommit(t))

	if !errors.Is(err, github.ErrTruncatedTree) {
		t.Fatalf("error is %v, want %v", err, github.ErrTruncatedTree)
	}
}

func TestListedFilesAtReportsACommitTheForgeDoesNotHold(t *testing.T) {
	server := answering(t, http.StatusNotFound, `{"message": "Not Found"}`)

	_, err := listingsAt(t, server).ListedFilesAt(context.Background(), policyHeadCommit(t))

	if !errors.Is(err, domain.ErrNoSuchCommit) || errors.Is(err, domain.ErrNoSuchPullRequest) {
		t.Fatalf("error is %v, want only %v", err, domain.ErrNoSuchCommit)
	}
}

func listingsAt(t *testing.T, server *httptest.Server) github.PolicyListings {
	t.Helper()

	listings, err := github.ParsePolicyListings(settingsAt(server))
	if err != nil {
		t.Fatalf("parse policy listings: %v", err)
	}
	return listings
}

func policyHeadCommit(t *testing.T) domain.PolicyCommit {
	t.Helper()

	at, err := domain.ParsePolicyCommit(policyRepositoryOf(t, readPolicyRepository), commitOf(t, policyHead))
	if err != nil {
		t.Fatalf("parse policy commit: %v", err)
	}
	return at
}

func commitOf(t *testing.T, stated string) domain.CommitSHA {
	t.Helper()

	commit, err := domain.ParseCommitSHA(stated)
	if err != nil {
		t.Fatalf("parse commit %q: %v", stated, err)
	}
	return commit
}
