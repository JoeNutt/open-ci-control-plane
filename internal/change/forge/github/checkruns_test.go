// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

const headSHA = "6dcb09b5b57875f334f61aebed695e2e4193db5e"

type wireOutput struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Text    string `json:"text"`
}

type wireCheckRun struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	HeadSHA    string     `json:"head_sha"`
	Status     string     `json:"status"`
	Conclusion string     `json:"conclusion"`
	Output     wireOutput `json:"output"`
}

type checkRunForge struct {
	t      *testing.T
	server *httptest.Server
	mutex  sync.Mutex
	runs   []wireCheckRun
}

func newCheckRunForge(t *testing.T) *checkRunForge {
	t.Helper()

	held := &checkRunForge{t: t}
	routes := http.NewServeMux()
	routes.HandleFunc("GET /repos/acme/payments/commits/{ref}/check-runs", held.list)
	routes.HandleFunc("POST /repos/acme/payments/check-runs", held.create)
	routes.HandleFunc("PATCH /repos/acme/payments/check-runs/{id}", held.update)
	held.server = forge(t, routes.ServeHTTP)
	return held
}

func (held *checkRunForge) list(writer http.ResponseWriter, request *http.Request) {
	held.mutex.Lock()
	defer held.mutex.Unlock()

	named := request.URL.Query().Get("check_name")
	matching := []wireCheckRun{}
	for _, run := range held.runs {
		if run.HeadSHA == request.PathValue("ref") && run.Name == named {
			matching = append(matching, run)
		}
	}
	held.answer(writer, http.StatusOK, map[string]any{
		"total_count": len(matching),
		"check_runs":  matching,
	})
}

func (held *checkRunForge) create(writer http.ResponseWriter, request *http.Request) {
	held.mutex.Lock()
	defer held.mutex.Unlock()

	var posted wireCheckRun
	if err := json.NewDecoder(request.Body).Decode(&posted); err != nil {
		held.t.Errorf("decode the posted check run: %v", err)
		writer.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	posted.ID = int64(len(held.runs) + 1)
	held.runs = append(held.runs, posted)
	held.answer(writer, http.StatusCreated, posted)
}

func (held *checkRunForge) update(writer http.ResponseWriter, request *http.Request) {
	held.mutex.Lock()
	defer held.mutex.Unlock()

	position := slices.IndexFunc(held.runs, func(run wireCheckRun) bool {
		return strconv.FormatInt(run.ID, 10) == request.PathValue("id")
	})
	if position < 0 {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	var patched wireCheckRun
	if err := json.NewDecoder(request.Body).Decode(&patched); err != nil {
		held.t.Errorf("decode the patched check run: %v", err)
		writer.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	updated := &held.runs[position]
	updated.Status = patched.Status
	updated.Conclusion = patched.Conclusion
	updated.Output = patched.Output
	held.answer(writer, http.StatusOK, *updated)
}

func (held *checkRunForge) answer(writer http.ResponseWriter, status int, body any) {
	encoded, err := json.Marshal(body)
	if err != nil {
		held.t.Errorf("encode answer: %v", err)
	}
	writer.WriteHeader(status)
	write(held.t, writer, string(encoded))
}

func (held *checkRunForge) runsOn(sha string) []wireCheckRun {
	held.mutex.Lock()
	defer held.mutex.Unlock()

	var on []wireCheckRun
	for _, run := range held.runs {
		if run.HeadSHA == sha {
			on = append(on, run)
		}
	}
	return on
}

func checkRunsAt(t *testing.T, server *httptest.Server) github.CheckRuns {
	t.Helper()

	settings := wellFormedSettings()
	settings.BaseURL = server.URL
	settings.Client = server.Client()
	runs, err := github.ParseCheckRuns(settings)
	if err != nil {
		t.Fatalf("parse check runs: %v", err)
	}
	return runs
}

func securityRun() domain.CheckRun {
	return domain.CheckRun{
		Name:       "policy/security",
		Conclusion: domain.ConclusionFailure,
		Title:      "1 violation",
		Summary:    "secrets.no_plaintext refused the change",
		Text:       "config/prod.env: a credential is committed",
	}
}

func TestPublishCheckRunCreatesARunOnACommitThatHasNone(t *testing.T) {
	held := newCheckRunForge(t)
	run := securityRun()

	err := checkRunsAt(t, held.server).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headSHA, run)

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	held.mustHoldOnly(heldAs(1, run))
}

func heldAs(id int64, run domain.CheckRun) wireCheckRun {
	return wireCheckRun{
		ID:         id,
		Name:       run.Name,
		HeadSHA:    headSHA,
		Status:     "completed",
		Conclusion: string(run.Conclusion),
		Output:     wireOutput{Title: run.Title, Summary: run.Summary, Text: run.Text},
	}
}

func (held *checkRunForge) mustHoldOnly(want wireCheckRun) {
	held.t.Helper()

	if got := held.runsOn(headSHA); !slices.Equal(got, []wireCheckRun{want}) {
		held.t.Fatalf("the forge holds %+v, want only %+v", got, want)
	}
}

func TestPublishCheckRunUpdatesTheRunOfTheSameNameOnTheSameCommit(t *testing.T) {
	held := newCheckRunForge(t)
	runs := checkRunsAt(t, held.server)
	locator := locatorOf(t, "acme/payments", 412)
	first := securityRun()
	second := first
	second.Conclusion = domain.ConclusionSuccess
	second.Title = "no violations"
	second.Summary = "every security policy admitted the change"
	second.Text = ""

	for _, run := range []domain.CheckRun{first, second} {
		if err := runs.PublishCheckRun(context.Background(), locator, headSHA, run); err != nil {
			t.Fatalf("publish check run: %v", err)
		}
	}

	held.mustHoldOnly(heldAs(1, second))
}

func TestPublishCheckRunRefusesToChooseBetweenRunsOfTheSameName(t *testing.T) {
	held := newCheckRunForge(t)
	run := securityRun()
	seeded := []wireCheckRun{heldAs(1, run), heldAs(2, run)}
	held.runs = slices.Clone(seeded)

	err := checkRunsAt(t, held.server).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headSHA, run)

	if !errors.Is(err, github.ErrAmbiguousCheckRun) {
		t.Fatalf("error is %v, want %v", err, github.ErrAmbiguousCheckRun)
	}
	if got := held.runsOn(headSHA); !slices.Equal(got, seeded) {
		t.Fatalf("the forge holds %+v, want it untouched as %+v", got, seeded)
	}
}

func refusingEveryRequest(t *testing.T) *httptest.Server {
	t.Helper()

	return forge(t, func(writer http.ResponseWriter, request *http.Request) {
		t.Errorf("the adapter sent %s %s, but should have sent nothing",
			request.Method, request.URL)
		writer.WriteHeader(http.StatusTeapot)
	})
}

func TestPublishCheckRunRefusesAMalformedHeadSHAWithoutAsking(t *testing.T) {
	malformed := []string{
		"",
		"6dcb09b",
		"6DCB09B5B57875F334F61AEBED695E2E4193DB5E",
		headSHA + "0",
		headSHA[:39],
		"6dcb09b5b57875f334f61aebed695e2e4193db5g",
		"../../../../../user/repos?per_page=100000",
		"6dcb09b5b57875f334f61aebed695e2e/../../x",
		"6dcb09b5b57875f334f61aebed695e2e4193d%2f",
	}
	runs := checkRunsAt(t, refusingEveryRequest(t))

	for _, sha := range malformed {
		t.Run(sha, func(t *testing.T) {
			err := runs.PublishCheckRun(
				context.Background(), locatorOf(t, "acme/payments", 412), sha, securityRun())

			if !errors.Is(err, github.ErrMalformedHeadSHA) {
				t.Fatalf("error is %v, want %v", err, github.ErrMalformedHeadSHA)
			}
		})
	}
}
