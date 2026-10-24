// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

const (
	dispatchedWorkflow = "open-ci-tests.yml"
	dispatchedSuite    = "example.com/payments/internal/orders"
	dispatchedTest     = "TestRefund"
	alwaysRunSuite     = "example.com/payments/internal/domain"
	planRevision       = "c0ffee0000000000000000000000000000c0ffee"
	dispatchPath       = "/repos/acme/payments/actions/workflows/" + dispatchedWorkflow + "/dispatches"
)

type wirePlan struct {
	SchemaVersion   int           `json:"schema_version"`
	DecisionID      string        `json:"decisionId"`
	Repository      string        `json:"repository"`
	HeadSHA         string        `json:"headSha"`
	Tests           []wireTests   `json:"tests"`
	AlwaysRunSuites []string      `json:"alwaysRunSuites"`
	FullMatrix      bool          `json:"fullMatrix"`
	TriggerReason   string        `json:"triggerReason"`
	Shadow          bool          `json:"shadow"`
	BundleRevision  string        `json:"bundleRevision"`
	Mode            wireMode      `json:"mode"`
	Recorded        *wireRecorded `json:"recorded,omitempty"`
}

type wireTests struct {
	Package string   `json:"package"`
	Names   []string `json:"names"`
}

type wireRecorded struct {
	Selected  []wireTests `json:"selected"`
	Minimised []wireTests `json:"minimised"`
}

type wireMode struct {
	Selection    string `json:"selection"`
	Minimisation string `json:"minimisation"`
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
		Tests: []wireTests{{Package: dispatchedSuite, Names: []string{}}}, AlwaysRunSuites: []string{alwaysRunSuite},
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
		plan := planSerialisingTo(t, bound.characters)

		sent := sentPlanOf(t, plan)
		isReplaced := bound.trigger == "plan_too_large"
		if sent.TriggerReason != bound.trigger || sent.FullMatrix != isReplaced || (len(sent.Tests) == 0) != isReplaced {
			t.Errorf("%d characters: sent %q, full matrix %v, %d tests; want %q",
				bound.characters, sent.TriggerReason, sent.FullMatrix, len(sent.Tests), bound.trigger)
		}
	}
}

func TestAPlanNamingTestsIsSentByNameAtTheLimitAndAsWholePackagesOneCharacterOver(t *testing.T) {
	for _, bound := range []struct {
		characters int
		isNamed    bool
	}{
		{domain.MaxPlanCharacters, true},
		{domain.MaxPlanCharacters + 1, false},
	} {
		plan := namedPlanSerialisingTo(t, bound.characters)

		sent := sentPlanOf(t, plan)
		if sent.TriggerReason != "none" || sent.FullMatrix || len(sent.Tests) != 1 ||
			sent.Tests[0].Package != dispatchedSuite || (len(sent.Tests[0].Names) > 0) != bound.isNamed {
			t.Errorf("%d characters: sent %q, full matrix %v, tests %d; want %s selected, by name %v",
				bound.characters, sent.TriggerReason, sent.FullMatrix, len(sent.Tests), dispatchedSuite, bound.isNamed)
		}
	}
}

func TestAPlanNamingTestsWhoseWholePackagesFitIsSentWholeAndOneCharacterOverIsTheFullMatrix(t *testing.T) {
	for _, bound := range []struct {
		characters int
		trigger    string
	}{
		{domain.MaxPlanCharacters, "none"},
		{domain.MaxPlanCharacters + 1, "plan_too_large"},
	} {
		plan := planSerialisingTo(t, bound.characters)
		for index, entry := range plan.Tests {
			plan.Tests[index] = domain.NewPackageTests(entry.Package(), []domain.TopLevelTestName{testNamed(t, dispatchedTest)})
		}

		sent := sentPlanOf(t, plan)
		isReplaced := bound.trigger == "plan_too_large"
		if sent.TriggerReason != bound.trigger || sent.FullMatrix != isReplaced || (len(sent.Tests) == 0) != isReplaced ||
			slices.ContainsFunc(sent.Tests, func(entry wireTests) bool { return len(entry.Names) > 0 }) {
			t.Errorf("%d characters as whole packages: sent %q, full matrix %v, %d tests %+v; want %q with no names",
				bound.characters, sent.TriggerReason, sent.FullMatrix, len(sent.Tests), sent.Tests, bound.trigger)
		}
	}
}

func namedPlanSerialisingTo(t *testing.T, characters int) domain.TestPlan {
	t.Helper()

	const nameLength = 20
	nameOverhead := len(`"",`)
	plan := selectedPlan(t)
	toFill := characters - len(encodedWirePlanOf(t, plan)) + 1
	count, longer := toFill/(nameLength+nameOverhead), toFill%(nameLength+nameOverhead)
	names := make([]domain.TopLevelTestName, 0, count)
	for index := range count {
		padding := ""
		if index < longer {
			padding = "Z"
		}
		names = append(names, testNamed(t, fmt.Sprintf("Test%s%0*d", padding, nameLength-len("Test"), index)))
	}
	plan.Tests = []domain.PackageTests{domain.NewPackageTests(plan.Tests[0].Package(), names)}
	if got := len(encodedWirePlanOf(t, plan)); got != characters {
		t.Fatalf("built a plan of %d characters, want %d", got, characters)
	}
	return plan
}

func planSerialisingTo(t *testing.T, characters int) domain.TestPlan {
	t.Helper()

	entryOverhead := len(`{"package":"","names":[]},`)
	plan := selectedPlan(t)
	plan.Tests = nil
	toFill := characters - len(encodedWirePlanOf(t, plan)) + 1
	count := (toFill + domain.MaxGoImportPathBytes + entryOverhead - 1) / (domain.MaxGoImportPathBytes + entryOverhead)
	share, extra := toFill/count, toFill%count
	for index := range count {
		length := share - entryOverhead
		if index < extra {
			length++
		}
		name := fmt.Sprintf("example.com/payments/s%d/", index)
		pkg, err := domain.ParseGoImportPath(name + strings.Repeat("a", length-len(name)))
		if err != nil {
			t.Fatalf("parse package: %v", err)
		}
		plan.Tests = append(plan.Tests, domain.NewPackageTests(pkg, nil))
	}
	if got := len(encodedWirePlanOf(t, plan)); got != characters {
		t.Fatalf("built a plan of %d characters, want %d", got, characters)
	}
	return plan
}

func wireTestsOf(tests []domain.PackageTests) []wireTests {
	entries := []wireTests{}
	for _, entry := range tests {
		names := []string{}
		for _, name := range entry.Names() {
			names = append(names, name.String())
		}
		entries = append(entries, wireTests{Package: entry.Package().String(), Names: names})
	}
	return entries
}

func encodedWirePlanOf(t *testing.T, plan domain.TestPlan) []byte {
	t.Helper()

	sent := wirePlan{
		SchemaVersion: 1, DecisionID: plan.DecisionID.String(), Repository: plan.Repository.String(),
		HeadSHA: plan.Head.String(), Tests: wireTestsOf(plan.Tests), AlwaysRunSuites: []string{},
		FullMatrix: plan.FullMatrix, TriggerReason: string(plan.Trigger), BundleRevision: plan.BundleRevision.String(),
		Mode: wireMode{Selection: plan.Modes.Selection.String(), Minimisation: plan.Modes.Minimisation.String()},
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

func TestDispatchPlanRetriesADispatchTheForgeAnswersWithAServerErrorThenGivesUp(t *testing.T) {
	held := newDispatchForge(t)
	busy := refusingFirst(t, held.server.Config.Handler, serverErrors(domain.OutboundAttemptLimit)...)
	waited := &pauses{}

	err := github.PlanDispatchesPausingWith(dispatchesAt(t, busy.server), waited.record).DispatchPlan(
		context.Background(), selectedPlan(t))

	if !errors.Is(err, github.ErrRetriesExhausted) {
		t.Fatalf("error is %v, want %v", err, github.ErrRetriesExhausted)
	}
	if busy.answered != domain.OutboundAttemptLimit || len(waited.durations) != domain.OutboundAttemptLimit-1 {
		t.Errorf("asked %d times after %d waits, want %d asks", busy.answered, len(waited.durations), domain.OutboundAttemptLimit)
	}
}

func TestDispatchPlanReportsAFailedDispatchWithoutTheCredential(t *testing.T) {
	echoing := failingOn(t, failedStep{http.MethodPost, http.StatusUnprocessableEntity, `{"default_branch": "main"}`})

	err := dispatchesAt(t, echoing).DispatchPlan(context.Background(), selectedPlan(t))

	if !errors.Is(err, github.ErrUnexpectedStatus) {
		t.Fatalf("error is %v, want %v", err, github.ErrUnexpectedStatus)
	}
	if strings.Contains(err.Error(), forgeToken) {
		t.Errorf("error %q reveals the credential", err)
	}
}

func TestEveryForgeConnectionAdmitsTheCallTimeoutAndRefusesOneNanosecondOver(t *testing.T) {
	for _, bound := range []struct {
		timeout    time.Duration
		isRefusing bool
	}{
		{domain.OutboundCallTimeout, false},
		{domain.OutboundCallTimeout + time.Nanosecond, true},
	} {
		settings := wellFormedSettings()
		settings.Client = &http.Client{Timeout: bound.timeout}

		_, err := github.ParsePlanDispatches(settings)

		if errors.Is(err, github.ErrUnboundedClient) != bound.isRefusing {
			t.Errorf("timeout %v: error is %v, want refused %v", bound.timeout, err, bound.isRefusing)
		}
	}
}

func sentPlanOf(t *testing.T, plan domain.TestPlan) wirePlan {
	t.Helper()

	held := newDispatchForge(t)
	if err := dispatchesAt(t, held.server).DispatchPlan(context.Background(), plan); err != nil {
		t.Fatalf("dispatch plan: %v", err)
	}
	var sent wirePlan
	if err := json.Unmarshal([]byte(held.onlyDispatch().Inputs.Plan), &sent); err != nil {
		t.Fatalf("decode the plan: %v", err)
	}
	return sent
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
	pkg, err := domain.ParseGoImportPath(dispatchedSuite)
	if err != nil {
		t.Fatalf("parse package: %v", err)
	}
	revision, err := domain.ParseBundleRevision(planRevision)
	if err != nil {
		t.Fatalf("parse revision: %v", err)
	}
	return domain.TestPlan{
		DecisionID: decisionID, Repository: repository, Head: headCommit, Workflow: workflow,
		Suites: []domain.CoverageSuite{suite}, Tests: []domain.PackageTests{domain.NewPackageTests(pkg, nil)},
		AlwaysRun: []domain.GoImportPath{always}, Trigger: domain.PlanTriggerNone, BundleRevision: revision,
	}
}

func TestAShadowPlanIsDispatchedWithItsModesAndRecordedSets(t *testing.T) {
	plan := selectedPlan(t)
	named := []domain.PackageTests{domain.NewPackageTests(plan.Tests[0].Package(), []domain.TopLevelTestName{testNamed(t, dispatchedTest)})}
	plan.Recorded = domain.RecordedSets{IsRecorded: true, Selected: named, Minimised: plan.Tests}
	plan.Suites, plan.Tests, plan.FullMatrix = nil, nil, true
	plan.Modes = domain.PlanModes{Selection: domain.TestModeShadow, Minimisation: domain.TestModeShadow}

	sent := sentPlanOf(t, plan)
	want := wireRecorded{
		Selected:  []wireTests{{Package: dispatchedSuite, Names: []string{dispatchedTest}}},
		Minimised: []wireTests{{Package: dispatchedSuite, Names: []string{}}},
	}
	if !sent.Shadow || sent.Mode.Selection != "shadow" || sent.Mode.Minimisation != "shadow" || sent.Recorded == nil ||
		!reflect.DeepEqual(*sent.Recorded, want) {
		t.Errorf("dispatched %+v, want shadow with both modes and both recorded sets", sent)
	}
}

func testNamed(t *testing.T, stated string) domain.TopLevelTestName {
	t.Helper()

	name, err := domain.ParseTopLevelTestName(stated)
	if err != nil {
		t.Fatalf("parse test name: %v", err)
	}
	return name
}
