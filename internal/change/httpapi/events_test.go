// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/httpapi"
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

func TestAnAnswerThatCannotBeWrittenIsLogged(t *testing.T) {
	delivery := eventDelivery(`{}`)
	var logged bytes.Buffer

	httpapi.Routes(slog.New(slog.NewJSONHandler(&logged, nil))).ServeHTTP(hungUpForge{}, delivery)

	if !strings.Contains(logged.String(), `"level":"WARN"`) || !strings.Contains(logged.String(), hangUp.Error()) {
		t.Errorf("the unwritten answer was not logged with its cause: %q", logged.String())
	}
}

var hangUp = errors.New("the forge hung up")

type hungUpForge struct{}

func (hungUpForge) Header() http.Header { return http.Header{} }

func (hungUpForge) Write([]byte) (int, error) { return 0, hangUp }

func (hungUpForge) WriteHeader(int) {}

func eventDelivery(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/v1/forge/github/events", strings.NewReader(body))
}

func discardedLog() *slog.Logger { return slog.New(slog.DiscardHandler) }
