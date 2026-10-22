// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestPublishCheckRunRetriesAPostTheForgeAnswersWithAServerError(t *testing.T) {
	held := newCheckRunForge(t)
	busy := refusingFirst(t, held, refusal{status: http.StatusBadGateway})

	err := checkRunsAt(t, busy.server).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	held.onlyRun()
}

type refusal struct {
	status int
	header http.Header
}

type refusingForge struct {
	server   *httptest.Server
	mutex    sync.Mutex
	pending  []refusal
	refused  int
	answered int
}

func refusingFirst(t *testing.T, held *checkRunForge, refusals ...refusal) *refusingForge {
	t.Helper()

	busy := &refusingForge{pending: refusals}
	busy.server = forge(t, func(writer http.ResponseWriter, request *http.Request) {
		if answer, isRefused := busy.nextRefusal(); isRefused {
			for name, values := range answer.header {
				writer.Header()[name] = values
			}
			writer.WriteHeader(answer.status)
			return
		}
		held.server.Config.Handler.ServeHTTP(writer, request)
	})
	return busy
}

func (busy *refusingForge) nextRefusal() (refusal, bool) {
	busy.mutex.Lock()
	defer busy.mutex.Unlock()

	busy.answered++
	if len(busy.pending) == 0 {
		return refusal{}, false
	}
	next := busy.pending[0]
	busy.pending = busy.pending[1:]
	busy.refused++
	return next, true
}
