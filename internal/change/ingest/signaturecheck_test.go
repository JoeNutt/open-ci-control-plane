// SPDX-License-Identifier: GPL-3.0-or-later

package ingest_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"
)

const (
	deliveredBody = `{"action":"opened","number":412}`
	appSecret     = domain.WebhookSecret("s3cret")
	trueSignature = "sha256=51e90d780f2a25ff6328cb24733ed9c4b05b70202696d272ab38890b3f016416"
)

type acceptedDelivery struct {
	body   []byte
	served int
}

func (accepted *acceptedDelivery) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accepted.served++
	accepted.body, _ = io.ReadAll(r.Body)
	w.WriteHeader(http.StatusAccepted)
}

func delivery(signature string) *http.Request {
	asked := httptest.NewRequest(http.MethodPost, "/v1/forge/github/events",
		strings.NewReader(deliveredBody))
	if signature != "" {
		asked.Header.Set("X-Hub-Signature-256", signature)
	}
	return asked
}

func TestASignedDeliveryReachesWhatWasWaitingForIt(t *testing.T) {
	accepted := &acceptedDelivery{}
	checking := ingest.SignatureCheck{Accepted: accepted, Secret: appSecret}
	answered := httptest.NewRecorder()

	checking.ServeHTTP(answered, delivery(trueSignature))

	if accepted.served != 1 {
		t.Fatalf("the delivery was served onward %d times, want 1", accepted.served)
	}
	if string(accepted.body) != deliveredBody {
		t.Errorf("body seen onward is %q, want %q", accepted.body, deliveredBody)
	}
	if answered.Code != http.StatusAccepted {
		t.Errorf("status is %d, want %d", answered.Code, http.StatusAccepted)
	}
}
