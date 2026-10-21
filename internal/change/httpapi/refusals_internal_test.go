// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestARefusalWithoutItsParsedDeliveryIsLoggedAsAnError(t *testing.T) {
	var logged bytes.Buffer
	recording := refusals{log: slog.New(slog.NewJSONHandler(&logged, nil))}

	recording.RecordRefusal(
		httptest.NewRequest(http.MethodPost, "/", nil), errors.New("the reason given"))

	if !strings.Contains(logged.String(), `"level":"ERROR"`) {
		t.Errorf("an uncarried refusal was not logged as an error: %q", logged.String())
	}
	if !strings.Contains(logged.String(), `"reason":"the reason given"`) {
		t.Errorf("an uncarried refusal lost its reason: %q", logged.String())
	}
}

func TestTheAdmittedStageWithoutItsParsedDeliveryAnswersAFailure(t *testing.T) {
	var logged bytes.Buffer
	stage := admitted{
		allowed:   allowlist{41},
		answering: answers{log: slog.New(slog.DiscardHandler)},
		log:       slog.New(slog.NewJSONHandler(&logged, nil)),
	}
	answered := httptest.NewRecorder()

	stage.ServeHTTP(answered, httptest.NewRequest(http.MethodPost, "/",
		strings.NewReader(`{"action":"opened","installation":{"id":41}}`)))

	if answered.Code != http.StatusInternalServerError {
		t.Errorf("status is %d, want %d", answered.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(logged.String(), `"level":"ERROR"`) {
		t.Errorf("the wiring fault was not logged as an error: %q", logged.String())
	}
}
