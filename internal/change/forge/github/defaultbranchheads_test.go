// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

const (
	policyHead            = "637d049f40a67db2936d7c97668b07c531b4dcc5"
	readPolicyRepository  = "JoeNutt/open-ci-policy"
	policyRepositoryPath  = "/repos/" + readPolicyRepository
	policyBranchHeadsPath = policyRepositoryPath + "/git/ref/heads/"
	policyHeadAnswer      = `{"object": {"sha": "` + policyHead + `", "type": "commit"}}`
)

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
		policyRepositoryPath:           `{"default_branch": "main"}`,
		policyBranchHeadsPath + "main": policyHeadAnswer,
	})

	head, err := headsAt(t, server).DefaultBranchHeadOf(context.Background(), policyRepositoryOf(t, readPolicyRepository))

	if err != nil {
		t.Fatalf("default branch head: %v", err)
	}
	if head.String() != policyHead {
		t.Errorf("head is %s, want %s", head, policyHead)
	}
	want := []askedRequest{
		{method: http.MethodGet, escapedPath: policyRepositoryPath, authorization: "Bearer " + forgeToken},
		{method: http.MethodGet, escapedPath: policyBranchHeadsPath + "main", authorization: "Bearer " + forgeToken},
	}
	if !slices.Equal(asked, want) {
		t.Errorf("the adapter asked %+v, want %+v", asked, want)
	}
}

func TestDefaultBranchHeadOfAsksForTheBranchTheForgeNamedAsOnePathSegment(t *testing.T) {
	var asked []askedRequest
	const escapedBranch = "release%2F..%2Fv1%3Fref=x"
	server := recordingEach(t, &asked, map[string]string{
		policyRepositoryPath:                  `{"default_branch": "release/../v1?ref=x"}`,
		policyBranchHeadsPath + escapedBranch: policyHeadAnswer,
	})

	if _, err := headsAt(t, server).DefaultBranchHeadOf(context.Background(), policyRepositoryOf(t, readPolicyRepository)); err != nil {
		t.Fatalf("default branch head: %v", err)
	}
	if len(asked) != 2 || asked[1].escapedPath != policyBranchHeadsPath+escapedBranch || asked[1].rawQuery != "" {
		t.Errorf("the adapter asked %+v, want the branch as one escaped segment and no query", asked)
	}
}

func TestDefaultBranchHeadOfReportsARepositoryTheForgeDoesNotHold(t *testing.T) {
	server := answering(t, http.StatusNotFound, `{"message": "Not Found"}`)

	_, err := headsAt(t, server).DefaultBranchHeadOf(context.Background(), policyRepositoryOf(t, readPolicyRepository))

	if !errors.Is(err, domain.ErrNoSuchRepository) || errors.Is(err, domain.ErrNoSuchPullRequest) {
		t.Fatalf("error is %v, want only %v", err, domain.ErrNoSuchRepository)
	}
}

func TestDefaultBranchHeadOfRefusesARepositoryNamingNoDefaultBranchWithoutAskingForIt(t *testing.T) {
	var asked []askedRequest
	server := recordingEach(t, &asked, map[string]string{policyRepositoryPath: `{"default_branch": ""}`})

	_, err := headsAt(t, server).DefaultBranchHeadOf(context.Background(), policyRepositoryOf(t, readPolicyRepository))

	if !errors.Is(err, github.ErrNoDefaultBranch) {
		t.Fatalf("error is %v, want %v", err, github.ErrNoDefaultBranch)
	}
	if len(asked) != 1 {
		t.Errorf("the adapter asked %+v, want only the repository", asked)
	}
}

func TestDefaultBranchHeadOfReportsADefaultBranchTheForgeHoldsNoReferenceFor(t *testing.T) {
	var asked []askedRequest
	server := recordingEach(t, &asked, map[string]string{policyRepositoryPath: `{"default_branch": "main"}`})

	_, err := headsAt(t, server).DefaultBranchHeadOf(context.Background(), policyRepositoryOf(t, readPolicyRepository))

	if !errors.Is(err, domain.ErrNoSuchBranch) || errors.Is(err, domain.ErrNoSuchRepository) {
		t.Fatalf("error is %v, want only %v", err, domain.ErrNoSuchBranch)
	}
}

func TestDefaultBranchHeadOfCarriesNoTokenAForgeRefusalEchoes(t *testing.T) {
	server := forge(t, func(writer http.ResponseWriter, _ *http.Request) {
		echoRefusalCarrying(t, writer, forgeToken)
	})

	_, err := headsAt(t, server).DefaultBranchHeadOf(context.Background(), policyRepositoryOf(t, readPolicyRepository))

	if err == nil {
		t.Fatal("a refused read reported no error")
	}
	mustRevealNoneOf(t, "a refused default-branch read", err.Error(), []string{forgeToken})
}

func echoRefusalCarrying(t *testing.T, writer http.ResponseWriter, echoed string) {
	t.Helper()

	connection, buffered, err := http.NewResponseController(writer).Hijack()
	if err != nil {
		t.Fatalf("hijack: %v", err)
	}
	defer func() { _ = connection.Close() }()
	body := `{"message": "Bad credentials: ` + echoed + `"}`
	if _, err := buffered.WriteString("HTTP/1.1 401 Bad credentials " + echoed + "\r\nContent-Length: " +
		strconv.Itoa(len(body)) + "\r\nConnection: close\r\n\r\n" + body); err != nil {
		t.Errorf("write refusal: %v", err)
	}
	if err := buffered.Flush(); err != nil {
		t.Errorf("flush refusal: %v", err)
	}
}

func headsAt(t *testing.T, server *httptest.Server) github.DefaultBranchHeads {
	t.Helper()

	heads, err := github.ParseDefaultBranchHeads(settingsAt(server))
	if err != nil {
		t.Fatalf("parse default branch heads: %v", err)
	}
	return heads
}

func policyRepositoryOf(t *testing.T, stated string) domain.PolicyRepository {
	t.Helper()

	repository, err := domain.ParsePolicyRepository(stated)
	if err != nil {
		t.Fatalf("parse policy repository %q: %v", stated, err)
	}
	return repository
}
