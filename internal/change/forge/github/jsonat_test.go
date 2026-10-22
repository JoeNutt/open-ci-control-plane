// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"encoding/hex"
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
	holders := map[string]any{
		"credential": forgeCredential,
		"settings":   settings,
		"source":     sourceWith(t, settings),
		"check runs": checkRunsWith(t, settings),
	}

	for name, holder := range holders {
		mustNotRevealUnder(t, name, holder, everyFormatVerb)
	}
}

func TestFormattingAHolderNestedInAnUnexportedFieldDoesNotRevealItsCredential(t *testing.T) {
	settings := wellFormedSettings()
	nested := map[string]any{
		"settings":   struct{ s github.Settings }{settings},
		"credential": struct{ c github.Credential }{forgeCredential},
		"check runs": struct{ r github.CheckRuns }{checkRunsWith(t, settings)},
		"source":     struct{ s github.ChangeSource }{sourceWith(t, settings)},
	}

	for name, holder := range nested {
		mustNotRevealUnder(t, "nested "+name, holder, everyFormatVerb)
	}
}

func mustNotRevealUnder(t *testing.T, name string, holder any, verbs []string) {
	t.Helper()

	for _, verb := range verbs {
		mustRevealNoneOf(t, name+" under "+verb, fmt.Sprintf(verb, holder), []string{forgeToken})
	}
}

var everyFormatVerb = []string{
	"%v", "%+v", "%#v", "%T", "%s", "%q", "%x", "%X", "%d", "%t", "%o", "%O",
	"%b", "%c", "%U", "%e", "%E", "%f", "%F", "%g", "%G",
	"%10s", "%-20v", "%+d", "% x", "%#x", "%#q", "%08d", "%.3s", "%[1]v", "%p",
}

func reveals(printed, secret string) bool {
	compact := strings.ToLower(strings.ReplaceAll(printed, " ", ""))
	return strings.Contains(printed, secret) ||
		strings.Contains(compact, hex.EncodeToString([]byte(secret)))
}

func TestEveryRequestCarriesTheCredentialTheSourceHolds(t *testing.T) {
	server := forge(t, func(writer http.ResponseWriter, request *http.Request) {
		presented := request.Header.Get("Authorization")
		if presented != "Bearer "+forgeToken {
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
	settings := settingsAt(server)
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
