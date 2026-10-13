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
	source := sourceAt(t, server)

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, github.ErrUnexpectedStatus) {
		t.Fatalf("error is %v, want %v", err, github.ErrUnexpectedStatus)
	}
}

func TestFormattingASourceDoesNotRevealItsCredential(t *testing.T) {
	credential := github.Credential("s3cret")
	settings := github.Settings{
		BaseURL:          "https://api.example",
		Token:            credential,
		Client:           http.DefaultClient,
		MaxResponseBytes: 1 << 20,
	}
	source := sourceWith(t, settings)

	printed := fmt.Sprintf("%v %+v %#v %v %+v %#v %s %q %#v",
		source, source, source, settings, settings, settings, credential, credential, credential)

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
	source := sourceWith(t, github.Settings{
		BaseURL:          server.URL,
		Token:            "s3cret",
		Client:           server.Client(),
		MaxResponseBytes: 1 << 20,
	})

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("files changed: %v", err)
	}
}

func TestFilesChangedRefusesAnAnswerLargerThanTheResponseBound(t *testing.T) {
	server := serving(t, "/repos/acme/payments/pulls/412/files", `[
		{"filename": "internal/auth/middleware.go", "status": "modified"}
	]`)
	source := sourceWith(t, github.Settings{
		BaseURL:          server.URL,
		Token:            "s3cret",
		Client:           server.Client(),
		MaxResponseBytes: 8,
	})

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, github.ErrResponseTooLarge) {
		t.Fatalf("error is %v, want %v", err, github.ErrResponseTooLarge)
	}
}

func TestFilesChangedReportsAForgeThatKnowsNoSuchPullRequest(t *testing.T) {
	server := answering(t, http.StatusNotFound, `{"message": "Not Found"}`)
	source := sourceAt(t, server)

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, domain.ErrNoSuchPullRequest) {
		t.Fatalf("error is %v, want %v", err, domain.ErrNoSuchPullRequest)
	}
}
