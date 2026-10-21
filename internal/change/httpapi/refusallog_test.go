// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"
)

func TestEveryRefusalOfADeliveryIsOneLineNamingItAndTheReason(t *testing.T) {
	refusals := map[string]func() *http.Request{
		"a body over the bound": func() *http.Request {
			delivery := headedDelivery(&untouchedBody{})
			delivery.ContentLength = ingest.MaxDeliveryBytes + 1
			return delivery
		},
		"a body of no declared length": func() *http.Request {
			delivery := headedDelivery(&untouchedBody{})
			delivery.ContentLength = -1
			return delivery
		},
		"no signature": func() *http.Request {
			return headedDelivery(strings.NewReader(pullRequestPayload("opened")))
		},
		"an installation outside the allowlist": func() *http.Request {
			return signedDelivery("pull_request", pullRequestPayloadFrom(`,"installation":{"id":99}`))
		},
		"a body that is not JSON": func() *http.Request {
			return signedDelivery("pull_request", "not json")
		},
		"no parseable pull request": func() *http.Request {
			return signedDelivery("pull_request", `{"action":"opened","installation":{"id":41}}`)
		},
	}
	for form, delivery := range refusals {
		t.Run(form, func(t *testing.T) {
			lines := loggedLinesOf(t, delivery())

			if len(lines) != 1 {
				t.Fatalf("logged %d lines, want 1: %v", len(lines), lines)
			}
			if lines[0]["msg"] != "delivery refused" || lines[0]["deliveryId"] != statedDeliveryID || lines[0]["reason"] == "" {
				t.Errorf("the line is %v, want a refusal naming delivery %s and a reason", lines[0], statedDeliveryID)
			}
		})
	}
}

func TestAHeaderRefusalIsOneLineNamingTheReasonAndTheHeader(t *testing.T) {
	refused := map[string]struct{ header, value string }{
		"no event header":      {"X-GitHub-Event", ""},
		"no delivery header":   {"X-GitHub-Delivery", ""},
		"a malformed delivery": {"X-GitHub-Delivery", "../72d3162e"},
		"a malformed event":    {"X-GitHub-Event", "Pull_request"},
	}
	for form, stated := range refused {
		t.Run(form, func(t *testing.T) {
			delivery := headedDelivery(strings.NewReader(`{}`))
			delivery.Header.Set(stated.header, stated.value)

			lines := loggedLinesOf(t, delivery)

			if len(lines) != 1 {
				t.Fatalf("logged %d lines, want 1: %v", len(lines), lines)
			}
			if lines[0]["msg"] != "delivery refused" || lines[0]["reason"] == "" || lines[0]["field"] != stated.header {
				t.Errorf("the line is %v, want a refusal naming a reason and %s", lines[0], stated.header)
			}
		})
	}
}

func loggedLinesOf(t *testing.T, delivery *http.Request) []map[string]string {
	t.Helper()
	var logged bytes.Buffer
	routedWith(t, slog.New(slog.NewJSONHandler(&logged, nil)), &recordingGate{}).ServeHTTP(
		httptest.NewRecorder(), delivery)
	var lines []map[string]string
	for _, line := range bytes.Split(bytes.TrimSpace(logged.Bytes()), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var fields map[string]string
		if err := json.Unmarshal(line, &fields); err != nil {
			t.Fatalf("a logged line is not structured: %v: %q", err, line)
		}
		lines = append(lines, fields)
	}
	return lines
}
