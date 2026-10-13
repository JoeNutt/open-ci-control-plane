// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

func TestFilesChangedRefusesAnAnswerTheForgeDidNotServeWith200(t *testing.T) {
	server := answering(t, http.StatusInternalServerError, `[]`)
	source := sourceAt(server)

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, github.ErrUnexpectedStatus) {
		t.Fatalf("error is %v, want %v", err, github.ErrUnexpectedStatus)
	}
}

func TestFormattingASourceDoesNotRevealItsCredential(t *testing.T) {
	source := github.ChangeSource{BaseURL: "https://api.example", Token: "s3cret"}

	printed := fmt.Sprintf("%v %+v %s %q", source, source, source.Token, source.Token)

	if strings.Contains(printed, "s3cret") {
		t.Fatalf("a formatted source revealed its credential: %s", printed)
	}
}

func TestEveryRequestCarriesTheCredentialTheSourceHolds(t *testing.T) {
	server := forge(t, func(writer http.ResponseWriter, request *http.Request) {
		presented := request.Header.Get("Authorization")
		if presented != "Bearer s3cret" {
			t.Errorf("the adapter presented %q", presented)
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		write(t, writer, `[]`)
	})
	source := sourceAt(server)
	source.Token = "s3cret"

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("files changed: %v", err)
	}
}

func TestFilesChangedRefusesAnAnswerLargerThanTheResponseBound(t *testing.T) {
	server := serving(t, "/repos/acme/payments/pulls/412/files", `[
		{"filename": "internal/auth/middleware.go", "status": "modified"}
	]`)
	source := github.ChangeSource{
		BaseURL:          server.URL,
		Client:           server.Client(),
		MaxResponseBytes: 8,
	}

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, github.ErrResponseTooLarge) {
		t.Fatalf("error is %v, want %v", err, github.ErrResponseTooLarge)
	}
}

func TestFilesChangedReportsAForgeThatKnowsNoSuchPullRequest(t *testing.T) {
	server := answering(t, http.StatusNotFound, `{"message": "Not Found"}`)
	source := sourceAt(server)

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, domain.ErrNoSuchPullRequest) {
		t.Fatalf("error is %v, want %v", err, domain.ErrNoSuchPullRequest)
	}
}
