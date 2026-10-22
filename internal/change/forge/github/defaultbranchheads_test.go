// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

const policyHead = "637d049f40a67db2936d7c97668b07c531b4dcc5"

type askedRequest struct {
	method        string
	escapedPath   string
	rawQuery      string
	authorization string
}

func recordingEach(t *testing.T, asked *[]askedRequest, bodies map[string]string) *httptest.Server {
	t.Helper()

	return forge(t, func(writer http.ResponseWriter, request *http.Request) {
		*asked = append(*asked, askedRequest{
			method:        request.Method,
			escapedPath:   request.URL.EscapedPath(),
			rawQuery:      request.URL.RawQuery,
			authorization: request.Header.Get("Authorization"),
		})
		body, served := bodies[request.URL.EscapedPath()]
		if !served {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		write(t, writer, body)
	})
}

func TestDefaultBranchHeadOfReadsTheDefaultBranchThenItsCommitAsTheInstallation(t *testing.T) {
	var asked []askedRequest
	server := recordingEach(t, &asked, map[string]string{
		"/repos/JoeNutt/open-ci-policy":                    `{"default_branch": "main"}`,
		"/repos/JoeNutt/open-ci-policy/git/ref/heads/main": `{"object": {"sha": "` + policyHead + `", "type": "commit"}}`,
	})
	heads, err := github.ParseDefaultBranchHeads(settingsAt(server))
	if err != nil {
		t.Fatalf("parse default branch heads: %v", err)
	}

	head, err := heads.DefaultBranchHeadOf(context.Background(), policyRepositoryOf(t, "JoeNutt/open-ci-policy"))

	if err != nil {
		t.Fatalf("default branch head: %v", err)
	}
	if head.String() != policyHead {
		t.Errorf("head is %s, want %s", head, policyHead)
	}
	want := []askedRequest{
		{method: http.MethodGet, escapedPath: "/repos/JoeNutt/open-ci-policy", authorization: "Bearer " + forgeToken},
		{method: http.MethodGet, escapedPath: "/repos/JoeNutt/open-ci-policy/git/ref/heads/main", authorization: "Bearer " + forgeToken},
	}
	if !slices.Equal(asked, want) {
		t.Errorf("the adapter asked %+v, want %+v", asked, want)
	}
}

func TestDefaultBranchHeadOfAsksForTheBranchTheForgeNamedAsOnePathSegment(t *testing.T) {
	var asked []askedRequest
	const escapedBranch = "release%2F..%2Fv1%3Fref=x"
	server := recordingEach(t, &asked, map[string]string{
		"/repos/JoeNutt/open-ci-policy":                                `{"default_branch": "release/../v1?ref=x"}`,
		"/repos/JoeNutt/open-ci-policy/git/ref/heads/" + escapedBranch: `{"object": {"sha": "` + policyHead + `"}}`,
	})
	heads, err := github.ParseDefaultBranchHeads(settingsAt(server))
	if err != nil {
		t.Fatalf("parse default branch heads: %v", err)
	}

	if _, err := heads.DefaultBranchHeadOf(context.Background(), policyRepositoryOf(t, "JoeNutt/open-ci-policy")); err != nil {
		t.Fatalf("default branch head: %v", err)
	}
	if len(asked) != 2 || asked[1].escapedPath != "/repos/JoeNutt/open-ci-policy/git/ref/heads/"+escapedBranch || asked[1].rawQuery != "" {
		t.Errorf("the adapter asked %+v, want the branch as one escaped segment and no query", asked)
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
