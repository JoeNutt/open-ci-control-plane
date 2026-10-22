// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"net/http"
	"testing"
)

const baseSHA = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4"

type askedForPullRequest struct {
	method        string
	escapedPath   string
	rawQuery      string
	authorization string
}

func TestBaseCommitOfReadsThePullRequestsBaseCommitAsTheInstallation(t *testing.T) {
	var asked askedForPullRequest
	server := forge(t, func(writer http.ResponseWriter, request *http.Request) {
		asked = askedForPullRequest{
			method:        request.Method,
			escapedPath:   request.URL.EscapedPath(),
			rawQuery:      request.URL.RawQuery,
			authorization: request.Header.Get("Authorization"),
		}
		write(t, writer, `{"number": 412, "base": {"ref": "main", "sha": "`+baseSHA+`"}}`)
	})

	base, err := sourceAt(t, server).BaseCommitOf(context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("base commit of: %v", err)
	}
	if base.String() != baseSHA {
		t.Errorf("base commit is %q, want %q", base, baseSHA)
	}
	want := askedForPullRequest{
		method:        http.MethodGet,
		escapedPath:   "/repos/acme/payments/pulls/412",
		authorization: "Bearer " + forgeToken,
	}
	if asked != want {
		t.Errorf("the adapter asked %+v, want %+v", asked, want)
	}
}
