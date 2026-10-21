// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/extract/golang"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

func TestBuildDocumentReadsTheForgeOverItsAPI(t *testing.T) {
	answers := map[string]string{
		"/repos/acme/payments/pulls/412": `{
			"number": 412,
			"draft": false,
			"base": {"ref": "main"},
			"head": {"sha": "` + recordedHeadSHA + `"},
			"user": {"login": "jo", "type": "User"},
			"author_association": "MEMBER",
			"labels": [{"name": "backend"}]
		}`,
		"/repos/acme/payments/pulls/412/reviews": `[
			{"user": {"login": "sam", "type": "User"}, "state": "APPROVED"}
		]`,
		"/repos/acme/payments/pulls/412/files": `[
			{"filename": "internal/auth/middleware.go", "status": "modified",
			 "additions": 12, "deletions": 3, "patch": "@@ -1,3 +1,12 @@"}
		]`,
	}
	server := httptest.NewTLSServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			answer, served := answers[request.URL.Path]
			if !served {
				t.Errorf("the adapter asked for %s", request.URL.Path)
				writer.WriteHeader(http.StatusNotFound)
				return
			}
			if _, written := writer.Write([]byte(answer)); written != nil {
				t.Errorf("write response: %v", written)
			}
		}))
	t.Cleanup(server.Close)
	client := server.Client()
	client.Timeout = 5 * time.Second

	source, err := github.ParseChangeSource(github.Settings{
		BaseURL:          server.URL,
		Credentials:      github.NewCredential("s3cret"),
		Client:           client,
		MaxResponseBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("parse change source: %v", err)
	}
	locator := locatorOf(t, "acme/payments", 412)
	useCase := application.BuildDocument{
		Source:    source,
		Contents:  recordedAtHead(t, locator, "internal/auth/middleware.go", sourceImportingContext),
		Extractor: golang.Extractor{},
		Limits:    domain.DeclaredLimits(),
	}

	document, err := useCase.Execute(context.Background(), locator)

	if err != nil {
		t.Fatalf("build document: %v", err)
	}
	if document.PR.Repository != "acme/payments" || document.PR.Number != 412 {
		t.Errorf("the document names %s#%d", document.PR.Repository, document.PR.Number)
	}
	if len(document.PR.Approvals) != 1 || document.PR.Approvals[0].Login != "sam" {
		t.Errorf("approvals are %+v", document.PR.Approvals)
	}
	want := domain.Totals{Files: 1, Additions: 12, Deletions: 3, PatchBytes: 16}
	if document.Diff.Totals != want {
		t.Errorf("totals are %+v, want %+v", document.Diff.Totals, want)
	}
}
