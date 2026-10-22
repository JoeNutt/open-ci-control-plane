// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

func TestPublishCheckRunRetriesAPostTheForgeAnswersWithAServerError(t *testing.T) {
	held := newCheckRunForge(t)
	busy := refusingFirst(t, held.server.Config.Handler, refusal{status: http.StatusBadGateway})

	err := checkRunsWaitingOn(t, busy.server, &pauses{}).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	held.onlyRun()
}

func TestPublishCheckRunStopsAtTheOutboundAttemptLimit(t *testing.T) {
	held := newCheckRunForge(t)
	busy := refusingFirst(t, held.server.Config.Handler, serverErrors(domain.OutboundAttemptLimit+1)...)

	err := checkRunsWaitingOn(t, busy.server, &pauses{}).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

	if !errors.Is(err, github.ErrForgeUnavailable) || !errors.Is(err, github.ErrRetriesExhausted) {
		t.Fatalf("error is %v, want %v and %v", err, github.ErrForgeUnavailable, github.ErrRetriesExhausted)
	}
	if busy.answered != domain.OutboundAttemptLimit {
		t.Errorf("the forge was asked %d times, want %d", busy.answered, domain.OutboundAttemptLimit)
	}
}

func TestPublishCheckRunSucceedsOnItsLastAllowedAttempt(t *testing.T) {
	held := newCheckRunForge(t)
	busy := refusingFirst(t, held.server.Config.Handler, serverErrors(domain.OutboundAttemptLimit-1)...)

	err := checkRunsWaitingOn(t, busy.server, &pauses{}).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	held.onlyRun()
}

func TestPublishCheckRunBacksOffDoublingTheWaitBeforeEachRetry(t *testing.T) {
	held := newCheckRunForge(t)
	busy := refusingFirst(t, held.server.Config.Handler, serverErrors(3)...)
	var waited pauses

	err := checkRunsWaitingOn(t, busy.server, &waited).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	want := []time.Duration{github.FirstRetryPause, 2 * github.FirstRetryPause, 4 * github.FirstRetryPause}
	if !slices.Equal(waited.durations, want) {
		t.Errorf("waited %v before the retries, want %v", waited.durations, want)
	}
}

func TestPublishCheckRunRetriesARateLimitedPost(t *testing.T) {
	limits := map[string]refusal{
		"too many requests": {status: http.StatusTooManyRequests},
		"primary rate limit": {
			status: http.StatusForbidden, header: http.Header{"X-Ratelimit-Remaining": {"0"}},
		},
		"secondary rate limit": {status: http.StatusForbidden, header: http.Header{"Retry-After": {"1"}}},
	}

	for form, limit := range limits {
		t.Run(form, func(t *testing.T) {
			held := newCheckRunForge(t)
			busy := refusingFirst(t, held.server.Config.Handler, limit)

			err := checkRunsWaitingOn(t, busy.server, &pauses{}).PublishCheckRun(
				context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

			if err != nil {
				t.Fatalf("publish check run: %v", err)
			}
			held.onlyRun()
		})
	}
}

func TestPublishCheckRunNeverRetriesAClientErrorThatIsNotARateLimit(t *testing.T) {
	refused := map[string]refusal{
		"forbidden":            {status: http.StatusForbidden},
		"unprocessable entity": {status: http.StatusUnprocessableEntity},
		"rate limit not spent": {
			status: http.StatusForbidden, header: http.Header{"X-Ratelimit-Remaining": {"4999"}},
		},
	}

	for form, answer := range refused {
		t.Run(form, func(t *testing.T) {
			busy := refusingFirst(t, newCheckRunForge(t).server.Config.Handler, answer)

			err := checkRunsWaitingOn(t, busy.server, &pauses{}).PublishCheckRun(
				context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

			if err == nil || errors.Is(err, github.ErrForgeUnavailable) {
				t.Errorf("error is %v, want a refusal that is not %v", err, github.ErrForgeUnavailable)
			}
			if busy.answered != 1 {
				t.Errorf("the forge was asked %d times, want once", busy.answered)
			}
		})
	}
}

func TestPublishCheckRunWaitsAsLongAsTheForgeAsksBeforeRetrying(t *testing.T) {
	held := newCheckRunForge(t)
	const askedSeconds = 30
	busy := refusingFirst(t, held.server.Config.Handler, refusal{
		status: http.StatusTooManyRequests, header: http.Header{"Retry-After": {strconv.Itoa(askedSeconds)}},
	})
	var waited pauses

	err := checkRunsWaitingOn(t, busy.server, &waited).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	if want := []time.Duration{askedSeconds * time.Second}; !slices.Equal(waited.durations, want) {
		t.Errorf("waited %v, want %v", waited.durations, want)
	}
}

func TestPublishCheckRunGivesUpRatherThanWaitPastTheRetryWindow(t *testing.T) {
	beyondWindow := strconv.Itoa(int(domain.OutboundRetryWindow/time.Second) + 1)
	busy := refusingFirst(t, newCheckRunForge(t).server.Config.Handler, refusal{
		status: http.StatusTooManyRequests, header: http.Header{"Retry-After": {beyondWindow}},
	})
	var waited pauses

	err := checkRunsWaitingOn(t, busy.server, &waited).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

	if !errors.Is(err, github.ErrForgeUnavailable) {
		t.Fatalf("error is %v, want %v", err, github.ErrForgeUnavailable)
	}
	if len(waited.durations) != 0 || busy.answered != 1 {
		t.Errorf("waited %v and asked %d times, want no wait and one ask", waited.durations, busy.answered)
	}
}

func TestPublishCheckRunReportsExhaustedRetriesWithoutTheCredential(t *testing.T) {
	echoing := failingOn(t, failedStep{http.MethodGet, http.StatusServiceUnavailable, noRunsListed})

	err := checkRunsWaitingOn(t, echoing, &pauses{}).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

	if !errors.Is(err, github.ErrRetriesExhausted) {
		t.Fatalf("error is %v, want %v", err, github.ErrRetriesExhausted)
	}
	if strings.Contains(err.Error(), forgeToken) {
		t.Errorf("error %q reveals the credential", err)
	}
}

type pauses struct {
	durations []time.Duration
}

func (waited *pauses) record(_ context.Context, pause time.Duration) error {
	waited.durations = append(waited.durations, pause)
	return nil
}

func checkRunsWaitingOn(t *testing.T, server *httptest.Server, waited *pauses) github.CheckRuns {
	t.Helper()

	return github.CheckRunsPausingWith(checkRunsAt(t, server), waited.record)
}

func serverErrors(count int) []refusal {
	refusals := make([]refusal, count)
	for index := range refusals {
		refusals[index] = refusal{status: http.StatusServiceUnavailable}
	}
	return refusals
}

type refusal struct {
	status int
	header http.Header
}

type refusingForge struct {
	server   *httptest.Server
	mutex    sync.Mutex
	pending  []refusal
	answered int
}

func refusingFirst(t *testing.T, next http.Handler, refusals ...refusal) *refusingForge {
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
		next.ServeHTTP(writer, request)
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
	return next, true
}
