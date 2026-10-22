// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestDispatchPlanSendsAPlanAtTheCharacterLimitAndReplacesOneCharacterOver(t *testing.T) {
	for _, bound := range []struct {
		characters int
		trigger    string
	}{
		{domain.MaxPlanCharacters, "none"},
		{domain.MaxPlanCharacters + 1, "plan_too_large"},
	} {
		held := newDispatchForge(t)
		plan := planSerialisingTo(t, bound.characters)

		if err := dispatchesAt(t, held.server).DispatchPlan(context.Background(), plan); err != nil {
			t.Fatalf("dispatch plan: %v", err)
		}

		var sent wirePlan
		if err := json.Unmarshal([]byte(held.onlyDispatch().Inputs.Plan), &sent); err != nil {
			t.Fatalf("decode the plan: %v", err)
		}
		isReplaced := bound.trigger == "plan_too_large"
		if sent.TriggerReason != bound.trigger || sent.FullMatrix != isReplaced || (len(sent.Suites) == 0) != isReplaced {
			t.Errorf("%d characters: sent %q, full matrix %v, %d suites; want %q",
				bound.characters, sent.TriggerReason, sent.FullMatrix, len(sent.Suites), bound.trigger)
		}
	}
}

func planSerialisingTo(t *testing.T, characters int) domain.TestPlan {
	t.Helper()

	const quotesAndComma = 3
	plan := selectedPlan(t)
	plan.Suites = nil
	toFill := characters - len(encodedWirePlanOf(t, plan)) + 1
	count := (toFill + domain.MaxGoImportPathBytes + quotesAndComma - 1) / (domain.MaxGoImportPathBytes + quotesAndComma)
	share, extra := toFill/count, toFill%count
	for index := range count {
		length := share - quotesAndComma
		if index < extra {
			length++
		}
		name := fmt.Sprintf("example.com/payments/s%d/", index)
		suite, err := domain.ParseCoverageSuite(name + strings.Repeat("a", length-len(name)))
		if err != nil {
			t.Fatalf("parse suite: %v", err)
		}
		plan.Suites = append(plan.Suites, suite)
	}
	if got := len(encodedWirePlanOf(t, plan)); got != characters {
		t.Fatalf("built a plan of %d characters, want %d", got, characters)
	}
	return plan
}

func encodedWirePlanOf(t *testing.T, plan domain.TestPlan) []byte {
	t.Helper()

	sent := wirePlan{
		SchemaVersion: 1, DecisionID: plan.DecisionID.String(), Repository: plan.Repository.String(),
		HeadSHA: plan.Head.String(), Suites: []string{}, AlwaysRunSuites: []string{},
		FullMatrix: plan.FullMatrix, TriggerReason: string(plan.Trigger), BundleRevision: plan.BundleRevision.String(),
	}
	for _, suite := range plan.Suites {
		sent.Suites = append(sent.Suites, suite.String())
	}
	for _, suite := range plan.AlwaysRun {
		sent.AlwaysRunSuites = append(sent.AlwaysRunSuites, suite.String())
	}
	encoded, err := json.Marshal(sent)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return encoded
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
