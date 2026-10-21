// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/httpapi"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"
)

func TestADeliveryWithoutAnEventHeaderIsRefusedNamingIt(t *testing.T) {
	delivery := eventDelivery(`{}`)
	delivery.Header.Set("X-GitHub-Delivery", "72d3162e-cc78-11e3-81ab-4c9367dc0958")
	answered := httptest.NewRecorder()

	httpapi.Routes(discardedLog()).ServeHTTP(answered, delivery)

	if answered.Code != http.StatusBadRequest {
		t.Fatalf("status is %d, want %d", answered.Code, http.StatusBadRequest)
	}
	if !strings.Contains(answered.Body.String(), `"field":"X-GitHub-Event"`) {
		t.Errorf("the refusal does not name X-GitHub-Event: %s", answered.Body.String())
	}
}

func TestADeliveryWithoutADeliveryHeaderIsRefusedNamingIt(t *testing.T) {
	delivery := eventDelivery(`{}`)
	delivery.Header.Set("X-GitHub-Event", "pull_request")
	answered := httptest.NewRecorder()

	httpapi.Routes(discardedLog()).ServeHTTP(answered, delivery)

	if answered.Code != http.StatusBadRequest {
		t.Fatalf("status is %d, want %d", answered.Code, http.StatusBadRequest)
	}
	if !strings.Contains(answered.Body.String(), `"field":"X-GitHub-Delivery"`) {
		t.Errorf("the refusal does not name X-GitHub-Delivery: %s", answered.Body.String())
	}
}

func TestADeliveryWithoutEitherHeaderIsAnsweredWithOneRefusal(t *testing.T) {
	answered := httptest.NewRecorder()

	httpapi.Routes(discardedLog()).ServeHTTP(answered, eventDelivery(`{}`))

	decoding := json.NewDecoder(answered.Body)
	var first, second map[string]string
	if err := decoding.Decode(&first); err != nil {
		t.Fatalf("no refusal was written: %v", err)
	}
	if err := decoding.Decode(&second); !errors.Is(err, io.EOF) {
		t.Errorf("a second refusal followed the first: %v", second)
	}
}

func TestAMalformedHeaderIsRefusedNamingIt(t *testing.T) {
	malformed := map[string]struct{ header, value string }{
		"an event above 100 bytes":   {"X-GitHub-Event", strings.Repeat("e", 101)},
		"an event with a capital":    {"X-GitHub-Event", "Pull_request"},
		"an event with a line break": {"X-GitHub-Event", "pull\nrequest"},
		"a delivery above 200 bytes": {"X-GitHub-Delivery", strings.Repeat("d", 201)},
		"a delivery with a space":    {"X-GitHub-Delivery", "72d3162e cc78"},
		"a delivery with a slash":    {"X-GitHub-Delivery", "../72d3162e"},
	}
	for form, stated := range malformed {
		t.Run(form, func(t *testing.T) {
			delivery := eventDelivery(`{}`)
			delivery.Header.Set("X-GitHub-Event", "pull_request")
			delivery.Header.Set("X-GitHub-Delivery", "72d3162e-cc78-11e3-81ab-4c9367dc0958")
			delivery.Header.Set(stated.header, stated.value)
			answered := httptest.NewRecorder()

			httpapi.Routes(discardedLog()).ServeHTTP(answered, delivery)

			if answered.Code != http.StatusBadRequest {
				t.Fatalf("status is %d, want %d", answered.Code, http.StatusBadRequest)
			}
			if !strings.Contains(answered.Body.String(), `"field":"`+stated.header+`"`) {
				t.Errorf("the refusal does not name %s: %s", stated.header, answered.Body.String())
			}
		})
	}
}

func TestADeliveryDeclaringMoreThanTheBoundIsRefusedUnread(t *testing.T) {
	unread := &untouchedBody{}
	delivery := httptest.NewRequest(http.MethodPost, "/v1/forge/github/events", unread)
	delivery.Header.Set("X-GitHub-Event", "pull_request")
	delivery.Header.Set("X-GitHub-Delivery", "72d3162e-cc78-11e3-81ab-4c9367dc0958")
	delivery.ContentLength = ingest.MaxDeliveryBytes + 1
	answered := httptest.NewRecorder()

	httpapi.Routes(discardedLog()).ServeHTTP(answered, delivery)

	if answered.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status is %d, want %d", answered.Code, http.StatusRequestEntityTooLarge)
	}
	if unread.isRead {
		t.Error("the body of an oversized delivery was read")
	}
	if !strings.Contains(answered.Body.String(), `"limit":"`) {
		t.Errorf("the refusal does not name the limit: %s", answered.Body.String())
	}
}

type untouchedBody struct{ isRead bool }

func (body *untouchedBody) Read([]byte) (int, error) {
	body.isRead = true
	return 0, io.EOF
}

func TestAnAnswerThatCannotBeWrittenIsLogged(t *testing.T) {
	delivery := eventDelivery(`{}`)
	var logged bytes.Buffer

	httpapi.Routes(slog.New(slog.NewJSONHandler(&logged, nil))).ServeHTTP(hungUpForge{}, delivery)

	if !strings.Contains(logged.String(), `"level":"WARN"`) || !strings.Contains(logged.String(), errHungUp.Error()) {
		t.Errorf("the unwritten answer was not logged with its cause: %q", logged.String())
	}
}

var errHungUp = errors.New("the forge hung up")

type hungUpForge struct{}

func (hungUpForge) Header() http.Header { return http.Header{} }

func (hungUpForge) Write([]byte) (int, error) { return 0, errHungUp }

func (hungUpForge) WriteHeader(int) {}

func eventDelivery(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/v1/forge/github/events", strings.NewReader(body))
}

func discardedLog() *slog.Logger { return slog.New(slog.DiscardHandler) }
