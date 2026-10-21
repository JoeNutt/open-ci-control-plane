// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/httpapi"
)

func TestADeliveryWithoutAnEventHeaderIsRefusedNamingIt(t *testing.T) {
	delivery := httptest.NewRequest(http.MethodPost, "/v1/forge/github/events", strings.NewReader(`{}`))
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

func discardedLog() *slog.Logger { return slog.New(slog.DiscardHandler) }
