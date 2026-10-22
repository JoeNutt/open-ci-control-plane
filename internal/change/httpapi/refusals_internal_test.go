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
