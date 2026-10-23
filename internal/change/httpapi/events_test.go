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
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/lcov"
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

func TestAnUnsignedOrMissignedDeliveryIsRefusedWithoutDetailAndUnevaluated(t *testing.T) {
	opened := pullRequestPayload("opened")
	refused := map[string]string{
		"no signature":                  "",
		"a signature by another secret": "sha256=" + strings.Repeat("ab", sha256.Size),
		"a signature over another body": signatureOf(pullRequestPayload("closed")),
	}
	for form, signature := range refused {
		t.Run(form, func(t *testing.T) {
			gate := &recordingGate{}
			delivery := signedDelivery("pull_request", opened)
			delivery.Header.Set("X-Hub-Signature-256", signature)
			answered := httptest.NewRecorder()

			routedWith(t, discardedLog(), gate).ServeHTTP(answered, delivery)

			if answered.Code != http.StatusUnauthorized {
				t.Errorf("status is %d, want %d", answered.Code, http.StatusUnauthorized)
			}
			if answered.Body.Len() != 0 {
				t.Errorf("the refusal carries detail: %q", answered.Body.String())
			}
			if len(gate.evaluated) != 0 {
				t.Errorf("the gate evaluated %v", gate.evaluated)
			}
		})
	}
}

func TestASignedDeliveryFromNoAllowlistedInstallationIsNotFoundAndUnevaluated(t *testing.T) {
	unlisted := map[string]string{
		"an installation outside the allowlist": `,"installation":{"id":99}`,
		"no installation":                       ``,
		"an installation with no ID":            `,"installation":{}`,
	}
	for form, installation := range unlisted {
		t.Run(form, func(t *testing.T) {
			gate := &recordingGate{}
			answered := httptest.NewRecorder()

			routedWith(t, discardedLog(), gate).ServeHTTP(answered,
				signedDelivery("pull_request", pullRequestPayloadFrom(installation)))

			if answered.Code != http.StatusNotFound {
				t.Errorf("status is %d, want %d", answered.Code, http.StatusNotFound)
			}
			if len(gate.evaluated) != 0 {
				t.Errorf("the gate evaluated %v", gate.evaluated)
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

func TestAnAllowlistedDeliveryThatCarriesNoDecisionIsAnsweredWithNoContent(t *testing.T) {
	unrouted := map[string]struct{ event, action string }{
		"another event":                {"issues", "opened"},
		"a review of the pull request": {"pull_request_review", "submitted"},
		"a closed pull request":        {"pull_request", "closed"},
		"a labelled pull request":      {"pull_request", "labeled"},
		"an edited pull request":       {"pull_request", "edited"},
	}
	for form, stated := range unrouted {
		t.Run(form, func(t *testing.T) {
			gate := &recordingGate{}
			answered := httptest.NewRecorder()

			routedWith(t, discardedLog(), gate).ServeHTTP(answered,
				signedDelivery(stated.event, pullRequestPayload(stated.action)))

			if answered.Code != http.StatusNoContent {
				t.Errorf("status is %d, want %d", answered.Code, http.StatusNoContent)
			}
			if len(gate.evaluated) != 0 {
				t.Errorf("the gate evaluated %v", gate.evaluated)
			}
		})
	}
}

func TestAnEvaluatedPullRequestActionIsHandedToTheGateAndAccepted(t *testing.T) {
	for _, action := range []string{"opened", "synchronize", "reopened"} {
		t.Run(action, func(t *testing.T) {
			gate := &recordingGate{}
			answered := httptest.NewRecorder()

			routedWith(t, discardedLog(), gate).ServeHTTP(answered,
				signedDelivery("pull_request", pullRequestPayload(action)))

			if answered.Code != http.StatusAccepted {
				t.Errorf("status is %d, want %d", answered.Code, http.StatusAccepted)
			}
			if len(gate.evaluated) != 1 || gate.evaluated[0] != gatedPullRequest(t) {
				t.Errorf("the gate evaluated %v, want only %v", gate.evaluated, gatedPullRequest(t))
			}
		})
	}
}

func TestADeliveryIsEvaluatedByTheGateOfItsOwnInstallationOnly(t *testing.T) {
	const otherInstallation = allowlistedInstallation + 1
	own, other := &recordingGate{}, &recordingGate{}
	settings := listenerSettings(discardedLog(), own)
	settings.Gates = map[int64]httpapi.ChangeGate{allowlistedInstallation: own, otherInstallation: other}
	answered := httptest.NewRecorder()

	parsedRoutes(t, settings).ServeHTTP(answered, signedDelivery("pull_request",
		pullRequestPayloadFrom(fmt.Sprintf(`,"installation":{"id":%d}`, otherInstallation))))

	if answered.Code != http.StatusAccepted {
		t.Errorf("status is %d, want %d", answered.Code, http.StatusAccepted)
	}
	if len(other.evaluated) != 1 || len(own.evaluated) != 0 {
		t.Errorf("installation %d's gate evaluated %v and installation %d's %v, want only the second once",
			allowlistedInstallation, own.evaluated, otherInstallation, other.evaluated)
	}
}

func TestTheAcceptanceNamesTheDeliveryAndTheDecisionTheGateWasGiven(t *testing.T) {
	gate := &recordingGate{}
	answered := httptest.NewRecorder()

	routedWith(t, discardedLog(), gate).ServeHTTP(answered,
		signedDelivery("pull_request", pullRequestPayload("opened")))

	var accepted struct {
		DeliveryID string `json:"deliveryId"`
		DecisionID string `json:"decisionId"`
		Accepted   bool   `json:"accepted"`
	}
	if err := json.Unmarshal(answered.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("the acceptance is not JSON: %v: %q", err, answered.Body.String())
	}
	if accepted.DeliveryID != statedDeliveryID || !accepted.Accepted {
		t.Errorf("the acceptance is %+v, want delivery %s accepted", accepted, statedDeliveryID)
	}
	if accepted.DecisionID != mintedDecisionID || len(gate.decided) != 1 || gate.decided[0].String() != mintedDecisionID {
		t.Errorf("the acceptance names decision %q and the gate was given %v, want %s for both",
			accepted.DecisionID, gate.decided, mintedDecisionID)
	}
}

func TestAFailedEvaluationIsNotAcceptedAndIsLoggedWithItsCause(t *testing.T) {
	gate := &recordingGate{failure: errors.New("the forge is unreachable")}
	var logged bytes.Buffer
	answered := httptest.NewRecorder()

	routedWith(t, slog.New(slog.NewJSONHandler(&logged, nil)), gate).ServeHTTP(answered,
		signedDelivery("pull_request", pullRequestPayload("opened")))

	if answered.Code != http.StatusInternalServerError {
		t.Errorf("status is %d, want %d", answered.Code, http.StatusInternalServerError)
	}
	if strings.Contains(answered.Body.String(), "unreachable") {
		t.Errorf("the answer carries internal detail: %s", answered.Body.String())
	}
	for _, named := range []string{statedDeliveryID, mintedDecisionID, "the forge is unreachable", `"level":"ERROR"`} {
		if !strings.Contains(logged.String(), named) {
			t.Errorf("the failure log does not name %s: %q", named, logged.String())
		}
	}
}

func TestADecisionIDThatCannotBeMintedEvaluatesNothing(t *testing.T) {
	gate := &recordingGate{}
	var logged bytes.Buffer
	settings := listenerSettings(slog.New(slog.NewJSONHandler(&logged, nil)), gate)
	settings.DecisionIDs = unmintableDecisions{}
	answered := httptest.NewRecorder()

	parsedRoutes(t, settings).ServeHTTP(answered, signedDelivery("pull_request", pullRequestPayload("opened")))

	if answered.Code != http.StatusInternalServerError {
		t.Errorf("status is %d, want %d", answered.Code, http.StatusInternalServerError)
	}
	if len(gate.evaluated) != 0 {
		t.Errorf("the gate evaluated %v with no decision ID", gate.evaluated)
	}
	if !strings.Contains(logged.String(), errUnmintable.Error()) || !strings.Contains(logged.String(), statedDeliveryID) {
		t.Errorf("the failure is not logged with its delivery and cause: %q", logged.String())
	}
}

func TestAPullRequestNamingNoParseablePullRequestIsRefusedUnevaluated(t *testing.T) {
	unparseable := map[string]string{
		"no number":         `{"action":"opened","repository":{"full_name":"JoeNutt/open-ci-control-plane"},"installation":{"id":41}}`,
		"no repository":     `{"action":"opened","number":7,"installation":{"id":41}}`,
		"a repository path": `{"action":"opened","number":7,"repository":{"full_name":"../etc"},"installation":{"id":41}}`,
	}
	for form, payload := range unparseable {
		t.Run(form, func(t *testing.T) {
			gate := &recordingGate{}
			answered := httptest.NewRecorder()

			routedWith(t, discardedLog(), gate).ServeHTTP(answered, signedDelivery("pull_request", payload))

			if answered.Code != http.StatusBadRequest {
				t.Errorf("status is %d, want %d", answered.Code, http.StatusBadRequest)
			}
			if len(gate.evaluated) != 0 {
				t.Errorf("the gate evaluated %v", gate.evaluated)
			}
			if !strings.Contains(answered.Body.String(), `"code":"unparseable_pull_request"`) {
				t.Errorf("the refusal does not say which pull request: %s", answered.Body.String())
			}
		})
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
	return parsedRoutes(t, listenerSettings(log, gate))
}

func listenerSettings(log *slog.Logger, gate *recordingGate) httpapi.Settings {
	return httpapi.Settings{
		Log:         log,
		Secret:      domain.WebhookSecret(webhookSecret),
		Gates:       map[int64]httpapi.ChangeGate{allowlistedInstallation: gate},
		DecisionIDs: mintedDecisions{},
		Clock:       time.Now,
		Coverage:    unexercisedCoverage{},
		Artifacts:   lcov.Artifacts{},
		Runs:        &runReports{},
	}
}

func parsedRoutes(t *testing.T, settings httpapi.Settings) http.Handler {
	t.Helper()
	routes, err := httpapi.ParseRoutes(settings)
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

type recordingGate struct {
	evaluated []domain.PullRequestLocator
	decided   []domain.DecisionID
	failure   error
}

func (gate *recordingGate) Execute(
	_ context.Context, locator domain.PullRequestLocator, id domain.DecisionID,
) error {
	gate.evaluated = append(gate.evaluated, locator)
	gate.decided = append(gate.decided, id)
	return gate.failure
}

const mintedDecisionID = "dec_0123456789abcdef0123456789abcdef"

type mintedDecisions struct{}

func (mintedDecisions) NextDecisionID() (domain.DecisionID, error) {
	return domain.ParseDecisionID(mintedDecisionID)
}

const (
	gatedRepository = "JoeNutt/open-ci-control-plane"
	gatedNumber     = 7
)

func pullRequestPayload(action string) string {
	return fmt.Sprintf(`{"action":%q,"number":%d,"repository":{"full_name":%q},"installation":{"id":%d}}`,
		action, gatedNumber, gatedRepository, allowlistedInstallation)
}

func pullRequestPayloadFrom(installation string) string {
	return fmt.Sprintf(`{"action":"opened","number":%d,"repository":{"full_name":%q}%s}`,
		gatedNumber, gatedRepository, installation)
}

func gatedPullRequest(t *testing.T) domain.PullRequestLocator {
	t.Helper()
	locator, err := domain.ParsePullRequestLocator(gatedRepository, gatedNumber)
	if err != nil {
		t.Fatal(err)
	}
	return locator
}

var errUnmintable = errors.New("no decision ID can be minted")

type unmintableDecisions struct{}

func (unmintableDecisions) NextDecisionID() (domain.DecisionID, error) {
	return domain.DecisionID{}, errUnmintable
}

func signatureOf(body string) string {
	signing := hmac.New(sha256.New, []byte(webhookSecret))
	signing.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(signing.Sum(nil))
}

func discardedLog() *slog.Logger { return slog.New(slog.DiscardHandler) }
