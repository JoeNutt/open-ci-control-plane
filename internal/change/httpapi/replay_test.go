// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAReplayedDeliveryIsAnsweredWithTheOriginalAcceptanceUnevaluated(t *testing.T) {
	gate := &recordingGate{}
	settings := listenerSettings(discardedLog(), gate)
	settings.DecisionIDs = &sequentialDecisions{}
	routes := parsedRoutes(t, settings)
	original := httptest.NewRecorder()
	replayed := httptest.NewRecorder()

	routes.ServeHTTP(original, signedDelivery("pull_request", pullRequestPayload("opened")))
	routes.ServeHTTP(replayed, signedDelivery("pull_request", pullRequestPayload("opened")))

	if replayed.Code != http.StatusAccepted || replayed.Body.String() != original.Body.String() {
		t.Errorf("the replay was answered %d %q, want the original %d %q",
			replayed.Code, replayed.Body.String(), original.Code, original.Body.String())
	}
	if len(gate.evaluated) != 1 {
		t.Errorf("the gate evaluated %d times, want once", len(gate.evaluated))
	}
}

func TestADeliveryReplayedADayLaterIsEvaluatedAgain(t *testing.T) {
	gate := &recordingGate{}
	clock := &steppedClock{now: time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)}
	settings := listenerSettings(discardedLog(), gate)
	settings.Clock = clock.read
	routes := parsedRoutes(t, settings)

	routes.ServeHTTP(httptest.NewRecorder(), signedDelivery("pull_request", pullRequestPayload("opened")))
	clock.now = clock.now.Add(24 * time.Hour)
	replayed := httptest.NewRecorder()
	routes.ServeHTTP(replayed, signedDelivery("pull_request", pullRequestPayload("opened")))

	if len(gate.evaluated) != 2 {
		t.Errorf("the gate evaluated %d times, want twice: the record should have lapsed", len(gate.evaluated))
	}
}

type steppedClock struct{ now time.Time }

func (clock *steppedClock) read() time.Time { return clock.now }

type sequentialDecisions struct{ minted int }

func (sequence *sequentialDecisions) NextDecisionID() (domain.DecisionID, error) {
	sequence.minted++
	return domain.ParseDecisionID(fmt.Sprintf("dec_%032x", sequence.minted))
}
