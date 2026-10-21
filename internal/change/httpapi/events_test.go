// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/httpapi"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"
)

func TestADeliveryWithoutAnEventHeaderIsRefusedNamingIt(t *testing.T) {
	delivery := eventDelivery(`{}`)
	delivery.Header.Set("X-GitHub-Delivery", statedDeliveryID)
	answered := answerTo(t, delivery)

	if answered.Code != http.StatusBadRequest {
		t.Fatalf("status is %d, want %d", answered.Code, http.StatusBadRequest)
	}
	if !strings.Contains(answered.Body.String(), `"field":"X-GitHub-Event"`) {
		t.Errorf("the refusal does not name X-GitHub-Event: %s", answered.Body.String())
	}
}

func TestADeliveryWithoutADeliveryHeaderIsRefusedNamingIt(t *testing.T) {
	delivery := eventDelivery(`{}`)
	delivery.Header.Set("X-GitHub-Event", statedEvent)
	answered := answerTo(t, delivery)

	if answered.Code != http.StatusBadRequest {
		t.Fatalf("status is %d, want %d", answered.Code, http.StatusBadRequest)
	}
	if !strings.Contains(answered.Body.String(), `"field":"X-GitHub-Delivery"`) {
		t.Errorf("the refusal does not name X-GitHub-Delivery: %s", answered.Body.String())
	}
}

func TestADeliveryWithoutEitherHeaderIsAnsweredWithOneRefusal(t *testing.T) {
	answered := answerTo(t, eventDelivery(`{}`))

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
			delivery := headedDelivery(strings.NewReader(`{}`))
			delivery.Header.Set(stated.header, stated.value)

			answered := answerTo(t, delivery)

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
	delivery := headedDelivery(unread)
	delivery.ContentLength = ingest.MaxDeliveryBytes + 1
	answered := answerTo(t, delivery)

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

func TestADeliveryDeclaringExactlyTheBoundIsReadAndChecked(t *testing.T) {
	delivery := headedDelivery(strings.NewReader(strings.Repeat(" ", ingest.MaxDeliveryBytes)))
	answered := answerTo(t, delivery)

	if answered.Code != http.StatusUnauthorized {
		t.Errorf("status is %d, want %d: a body at the bound was not read and checked",
			answered.Code, http.StatusUnauthorized)
	}
}

func TestADeliveryDeclaringNoLengthIsRefusedUnread(t *testing.T) {
	unread := &untouchedBody{}
	delivery := headedDelivery(unread)
	delivery.ContentLength = -1
	delivery.TransferEncoding = []string{"chunked"}
	answered := answerTo(t, delivery)

	if answered.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status is %d, want %d", answered.Code, http.StatusRequestEntityTooLarge)
	}
	if unread.isRead {
		t.Error("the body of a delivery declaring no length was read")
	}
	if !strings.Contains(answered.Body.String(), `"code":"undeclared_length"`) {
		t.Errorf("the refusal does not say the length was undeclared: %s", answered.Body.String())
	}
}

func TestAnUnsignedDeliveryIsRefusedWithoutDetail(t *testing.T) {
	answered := answerTo(t, headedDelivery(strings.NewReader(`{}`)))

	if answered.Code != http.StatusUnauthorized {
		t.Errorf("status is %d, want %d", answered.Code, http.StatusUnauthorized)
	}
	if answered.Body.Len() != 0 {
		t.Errorf("the refusal carries detail: %q", answered.Body.String())
	}
}

func TestACorrectlySignedDeliveryIsAdmitted(t *testing.T) {
	delivery := headedDelivery(strings.NewReader(`{}`))
	delivery.Header.Set("X-Hub-Signature-256", signatureOf(`{}`))
	answered := answerTo(t, delivery)

	if answered.Code == http.StatusUnauthorized {
		t.Error("a correctly signed delivery was refused")
	}
}

func TestASignedDeliveryFromNoAllowlistedInstallationIsNotFound(t *testing.T) {
	unlisted := map[string]string{
		"an installation outside the allowlist": `{"installation":{"id":99}}`,
		"no installation":                       `{}`,
		"an installation with no ID":            `{"installation":{}}`,
	}
	for form, payload := range unlisted {
		t.Run(form, func(t *testing.T) {
			answered := answerTo(t, signedDelivery(statedEvent, payload))

			if answered.Code != http.StatusNotFound {
				t.Errorf("status is %d, want %d", answered.Code, http.StatusNotFound)
			}
		})
	}
}

func TestASignedBodyThatIsNotJSONIsRefusedAsMalformed(t *testing.T) {
	answered := answerTo(t, signedDelivery(statedEvent, `not json`))

	if answered.Code != http.StatusBadRequest {
		t.Errorf("status is %d, want %d", answered.Code, http.StatusBadRequest)
	}
	if !strings.Contains(answered.Body.String(), `"code":"malformed_payload"`) {
		t.Errorf("the refusal does not say the payload is malformed: %s", answered.Body.String())
	}
}

func TestAnAllowlistedEventThatCarriesNoDecisionIsAnsweredWithNoContent(t *testing.T) {
	answered := answerTo(t, signedDelivery("issues", `{"installation":{"id":41}}`))

	if answered.Code != http.StatusNoContent {
		t.Errorf("status is %d, want %d", answered.Code, http.StatusNoContent)
	}
}

func TestAnOpenedPullRequestIsHandedToTheGateAndAccepted(t *testing.T) {
	gate := &recordingGate{}
	answered := httptest.NewRecorder()

	routedWith(t, discardedLog(), gate).ServeHTTP(answered,
		signedDelivery("pull_request", pullRequestPayload("opened")))

	if answered.Code != http.StatusAccepted {
		t.Errorf("status is %d, want %d", answered.Code, http.StatusAccepted)
	}
	if len(gate.evaluated) != 1 || gate.evaluated[0] != gatedPullRequest(t) {
		t.Errorf("the gate evaluated %v, want only %v", gate.evaluated, gatedPullRequest(t))
	}
}

func TestARefusedSignatureIsLoggedNamingTheDeliveryAndTheReason(t *testing.T) {
	var logged bytes.Buffer

	routed(t, slog.New(slog.NewJSONHandler(&logged, nil))).ServeHTTP(
		httptest.NewRecorder(), headedDelivery(strings.NewReader(`{}`)))

	var line map[string]string
	if err := json.Unmarshal(logged.Bytes(), &line); err != nil {
		t.Fatalf("the refusal is not one structured line: %v: %q", err, logged.String())
	}
	if line["deliveryId"] != statedDeliveryID {
		t.Errorf("the refusal names delivery %q, want %q", line["deliveryId"], statedDeliveryID)
	}
	if line["reason"] == "" {
		t.Errorf("the refusal states no reason: %q", logged.String())
	}
}

func TestAnAnswerThatCannotBeWrittenIsLogged(t *testing.T) {
	delivery := eventDelivery(`{}`)
	var logged bytes.Buffer

	routed(t, slog.New(slog.NewJSONHandler(&logged, nil))).ServeHTTP(hungUpForge{}, delivery)

	if !strings.Contains(logged.String(), `"level":"WARN"`) || !strings.Contains(logged.String(), errHungUp.Error()) {
		t.Errorf("the unwritten answer was not logged with its cause: %q", logged.String())
	}
}

var errHungUp = errors.New("the forge hung up")

type hungUpForge struct{}

func (hungUpForge) Header() http.Header { return http.Header{} }

func (hungUpForge) Write([]byte) (int, error) { return 0, errHungUp }

func (hungUpForge) WriteHeader(int) {}

type untouchedBody struct{ isRead bool }

func (body *untouchedBody) Read([]byte) (int, error) {
	body.isRead = true
	return 0, io.EOF
}

const (
	eventsPath       = "/v1/forge/github/events"
	statedEvent      = "pull_request"
	statedDeliveryID = "72d3162e-cc78-11e3-81ab-4c9367dc0958"
)

func headedDelivery(body io.Reader) *http.Request {
	delivery := httptest.NewRequest(http.MethodPost, eventsPath, body)
	delivery.Header.Set("X-GitHub-Event", statedEvent)
	delivery.Header.Set("X-GitHub-Delivery", statedDeliveryID)
	return delivery
}

func eventDelivery(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, eventsPath, strings.NewReader(body))
}

const (
	webhookSecret           = "the App's webhook secret"
	allowlistedInstallation = 41
)

func routed(t *testing.T, log *slog.Logger) http.Handler {
	t.Helper()
	return routedWith(t, log, &recordingGate{})
}

func routedWith(t *testing.T, log *slog.Logger, gate *recordingGate) http.Handler {
	t.Helper()
	routes, err := httpapi.ParseRoutes(httpapi.Settings{
		Log:           log,
		Secret:        domain.WebhookSecret(webhookSecret),
		Installations: []int64{allowlistedInstallation},
		Gate:          gate,
	})
	if err != nil {
		t.Fatal(err)
	}
	return routes
}

func signedDelivery(event, payload string) *http.Request {
	delivery := headedDelivery(strings.NewReader(payload))
	delivery.Header.Set("X-GitHub-Event", event)
	delivery.Header.Set("X-Hub-Signature-256", signatureOf(payload))
	return delivery
}

func answerTo(t *testing.T, delivery *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	answered := httptest.NewRecorder()
	routed(t, discardedLog()).ServeHTTP(answered, delivery)
	return answered
}

type recordingGate struct{ evaluated []domain.PullRequestLocator }

func (gate *recordingGate) Execute(
	_ context.Context, locator domain.PullRequestLocator, _ domain.DecisionID,
) error {
	gate.evaluated = append(gate.evaluated, locator)
	return nil
}

const (
	gatedRepository = "JoeNutt/open-ci-control-plane"
	gatedNumber     = 7
)

func pullRequestPayload(action string) string {
	return fmt.Sprintf(
		`{"action":%q,"number":%d,"repository":{"full_name":%q},"installation":{"id":%d}}`,
		action, gatedNumber, gatedRepository, allowlistedInstallation)
}

func gatedPullRequest(t *testing.T) domain.PullRequestLocator {
	t.Helper()
	locator, err := domain.ParsePullRequestLocator(gatedRepository, gatedNumber)
	if err != nil {
		t.Fatal(err)
	}
	return locator
}

func signatureOf(body string) string {
	signing := hmac.New(sha256.New, []byte(webhookSecret))
	signing.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(signing.Sum(nil))
}

func discardedLog() *slog.Logger { return slog.New(slog.DiscardHandler) }
