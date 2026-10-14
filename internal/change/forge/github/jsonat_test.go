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

func TestFormattingAnAdapterDoesNotRevealItsCredential(t *testing.T) {
	settings := wellFormedSettings()
	runs, err := github.ParseCheckRuns(settings)
	if err != nil {
		t.Fatalf("parse check runs: %v", err)
	}
	holders := map[string]any{
		"credential": forgeCredential,
		"settings":   settings,
		"source":     sourceWith(t, settings),
		"check runs": runs,
	}

	for name, holder := range holders {
		for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			printed := fmt.Sprintf(verb, holder)
			if strings.Contains(printed, string(forgeCredential)) {
				t.Errorf("%s under %s revealed its credential: %s", name, verb, printed)
			}
		}
	}
}

func TestEveryRequestCarriesTheCredentialTheSourceHolds(t *testing.T) {
	server := forge(t, func(writer http.ResponseWriter, request *http.Request) {
		presented := request.Header.Get("Authorization")
		if presented != "Bearer "+string(forgeCredential) {
			t.Errorf("the adapter presented %q", presented)
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		write(t, writer, `[]`)
	})
	source := sourceAt(t, server)

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("files changed: %v", err)
	}
}

func TestFilesChangedRefusesAnAnswerLargerThanTheResponseBound(t *testing.T) {
	server := serving(t, "/repos/acme/payments/pulls/412/files", `[
		{"filename": "internal/auth/middleware.go", "status": "modified"}
	]`)
	settings := wellFormedSettings()
	settings.BaseURL = server.URL
	settings.Client = server.Client()
	settings.MaxResponseBytes = 8

	_, err := sourceWith(t, settings).FilesChanged(
		context.Background(), locatorOf(t, "acme/payments", 412))

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
