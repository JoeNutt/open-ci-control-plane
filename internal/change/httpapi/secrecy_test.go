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
)

const bodyContent = "content-of-a-changed-file"

func TestNoLogLineCarriesTheSecretASignatureOrTheBody(t *testing.T) {
	carried := `{"action":"opened","number":7,"repository":{"full_name":"JoeNutt/open-ci-control-plane"},` +
		`"installation":{"id":41},"pull_request":{"body":"` + bodyContent + `"}}`
	paths := map[string]struct {
		delivery func() *http.Request
		gate     *recordingGate
	}{
		"accepted":          {func() *http.Request { return signedDelivery("pull_request", carried) }, &recordingGate{}},
		"evaluation failed": {func() *http.Request { return signedDelivery("pull_request", carried) }, &recordingGate{failure: errors.New("refused")}},
		"missigned":         {func() *http.Request { return missigned(carried) }, &recordingGate{}},
		"not allowlisted": {func() *http.Request {
			return signedDelivery("pull_request", strings.Replace(carried, ":41}", ":99}", 1))
		}, &recordingGate{}},
		"no parseable pull request": {func() *http.Request {
			return signedDelivery("pull_request", strings.Replace(carried, `"number":7,`, "", 1))
		}, &recordingGate{}},
		"not JSON": {func() *http.Request { return signedDelivery("pull_request", bodyContent) }, &recordingGate{}},
	}
	for form, path := range paths {
		t.Run(form, func(t *testing.T) {
			var logged bytes.Buffer
			delivery := path.delivery()
			signature := delivery.Header.Get("X-Hub-Signature-256")

			routedWith(t, slog.New(slog.NewJSONHandler(&logged, nil)), path.gate).ServeHTTP(
				httptest.NewRecorder(), delivery)

			for name, secret := range map[string]string{
				"the webhook secret": webhookSecret, "the signature": signature, "the body": bodyContent,
			} {
				if strings.Contains(logged.String(), secret) {
					t.Errorf("a log line carries %s: %q", name, logged.String())
				}
			}
		})
	}
}

func missigned(payload string) *http.Request {
	delivery := signedDelivery("pull_request", payload)
	delivery.Header.Set("X-Hub-Signature-256", signatureOf(payload+" "))
	return delivery
}
