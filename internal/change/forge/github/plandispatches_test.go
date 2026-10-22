// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

const (
	dispatchedWorkflow = "open-ci-tests.yml"
	dispatchedSuite    = "example.com/payments/internal/orders"
	alwaysRunSuite     = "example.com/payments/internal/domain"
	planRevision       = "c0ffee0000000000000000000000000000c0ffee"
	dispatchPath       = "/repos/acme/payments/actions/workflows/" + dispatchedWorkflow + "/dispatches"
)

type wirePlan struct {
	SchemaVersion   int      `json:"schema_version"`
	DecisionID      string   `json:"decisionId"`
	Repository      string   `json:"repository"`
	HeadSHA         string   `json:"headSha"`
	Suites          []string `json:"suites"`
	AlwaysRunSuites []string `json:"alwaysRunSuites"`
	FullMatrix      bool     `json:"fullMatrix"`
	TriggerReason   string   `json:"triggerReason"`
	Shadow          bool     `json:"shadow"`
	BundleRevision  string   `json:"bundleRevision"`
}

type wireDispatch struct {
	Ref    string `json:"ref"`
	Inputs struct {
		Plan string `json:"plan"`
	} `json:"inputs"`
}

func TestDispatchPlanPostsTheC9PlanAsOneInputOnTheDefaultBranch(t *testing.T) {
	held := newDispatchForge(t)

	dispatched := dispatchesAt(t, held.server).DispatchPlan(context.Background(), selectedPlan(t))

	if dispatched != nil {
		t.Fatalf("dispatch plan: %v", dispatched)
	}
	sent := held.onlyDispatch()
	if sent.Ref != "main" {
		t.Errorf("ref is %q, want the default branch %q", sent.Ref, "main")
	}
	var plan wirePlan
	if err := json.Unmarshal([]byte(sent.Inputs.Plan), &plan); err != nil {
		t.Fatalf("the plan input is not C-9 JSON: %v", err)
	}
	want := wirePlan{
		SchemaVersion: 1, DecisionID: decisionText, Repository: "acme/payments", HeadSHA: headSHA,
		Suites: []string{dispatchedSuite}, AlwaysRunSuites: []string{alwaysRunSuite},
		TriggerReason: "none", BundleRevision: planRevision,
	}
	if !equalWirePlans(plan, want) {
		t.Errorf("dispatched %+v, want %+v", plan, want)
	}
}

func equalWirePlans(one, other wirePlan) bool {
	encodedOne, errOne := json.Marshal(one)
	encodedOther, errOther := json.Marshal(other)
	return errOne == nil && errOther == nil && string(encodedOne) == string(encodedOther)
}

type dispatchForge struct {
	t          *testing.T
	server     *httptest.Server
	mutex      sync.Mutex
	dispatches []wireDispatch
}

func newDispatchForge(t *testing.T) *dispatchForge {
	t.Helper()

	held := &dispatchForge{t: t}
	routes := http.NewServeMux()
	routes.HandleFunc("GET /repos/acme/payments", func(writer http.ResponseWriter, _ *http.Request) {
		write(t, writer, `{"default_branch": "main"}`)
	})
	routes.HandleFunc("POST "+dispatchPath, held.dispatch)
	held.server = forge(t, routes.ServeHTTP)
	return held
}

func (held *dispatchForge) dispatch(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		held.t.Errorf("read the dispatch: %v", err)
	}
	var sent wireDispatch
	if err := json.Unmarshal(body, &sent); err != nil {
		held.t.Errorf("decode the dispatch: %v", err)
	}
	held.mutex.Lock()
	held.dispatches = append(held.dispatches, sent)
	held.mutex.Unlock()
	writer.WriteHeader(http.StatusNoContent)
}

func (held *dispatchForge) onlyDispatch() wireDispatch {
	held.t.Helper()

	held.mutex.Lock()
	defer held.mutex.Unlock()
	if len(held.dispatches) != 1 {
		held.t.Fatalf("the forge received %d dispatches, want 1", len(held.dispatches))
	}
	return held.dispatches[0]
}

func dispatchesAt(t *testing.T, server *httptest.Server) github.PlanDispatches {
	t.Helper()

	dispatches, err := github.ParsePlanDispatches(settingsAt(server))
	if err != nil {
		t.Fatalf("parse plan dispatches: %v", err)
	}
	return dispatches
}

func selectedPlan(t *testing.T) domain.TestPlan {
	t.Helper()

	repository, err := domain.ParseGovernedRepository("acme/payments")
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}
	workflow, err := domain.ParseWorkflowFile(dispatchedWorkflow)
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	suite, err := domain.ParseCoverageSuite(dispatchedSuite)
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	always, err := domain.ParseGoImportPath(alwaysRunSuite)
	if err != nil {
		t.Fatalf("parse always-run suite: %v", err)
	}
	revision, err := domain.ParseBundleRevision(planRevision)
	if err != nil {
		t.Fatalf("parse revision: %v", err)
	}
	return domain.TestPlan{
		DecisionID: decisionID, Repository: repository, Head: headCommit, Workflow: workflow,
		Suites: []domain.CoverageSuite{suite}, AlwaysRun: []domain.GoImportPath{always},
		Trigger: domain.PlanTriggerNone, BundleRevision: revision,
	}
}
