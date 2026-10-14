// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

const headSHA = "6dcb09b5b57875f334f61aebed695e2e4193db5e"

var headCommit = func() domain.CommitSHA {
	head, err := domain.ParseCommitSHA(headSHA)
	if err != nil {
		panic(err)
	}
	return head
}()

const otherAppID = 99

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
	App        wireApp    `json:"app"`
}

type wireApp struct {
	ID int64 `json:"id"`
}

type checkRunForge struct {
	t      *testing.T
	server *httptest.Server
	mutex  sync.Mutex
	runs   []wireCheckRun

	ignoresAppID bool
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
		if run.HeadSHA == request.PathValue("ref") && run.Name == named &&
			held.isOwnedAsAsked(run, request) {
			matching = append(matching, run)
		}
	}
	held.answer(writer, http.StatusOK, map[string]any{
		"total_count": len(matching),
		"check_runs":  matching,
	})
}

func (held *checkRunForge) isOwnedAsAsked(run wireCheckRun, request *http.Request) bool {
	asked := request.URL.Query().Get("app_id")
	if asked == "" || held.ignoresAppID {
		return true
	}
	return strconv.FormatInt(run.App.ID, 10) == asked
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
	posted.App = wireApp{ID: forgeAppID}
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

	return checkRunsWith(t, settingsAt(server))
}

func checkRunsWith(t *testing.T, settings github.Settings) github.CheckRuns {
	t.Helper()

	runs, err := github.ParseCheckRuns(settings)
	if err != nil {
		t.Fatalf("parse check runs: %v", err)
	}
	return runs
}

const (
	securityRuleID  = "secrets.no_plaintext"
	securityMessage = "config/prod.env: a credential is committed"
	securityTeam    = "security-platform"
	securityDocsURL = "https://docs.example.com/secrets"
	decisionText    = "dec_0123456789abcdef0123456789abcdef"
)

func securityRun() domain.CheckRun {
	return domain.CheckRun{
		Name:       "policy/security",
		Conclusion: domain.ConclusionFailure,
		Title:      "1 violation",
		Headline:   securityRuleID + " refused the change",
		DecisionID: decisionID,
		Violations: []domain.CheckRunViolation{{
			RuleID: securityRuleID, Message: securityMessage,
			OwningTeam: securityTeam, DocsURL: securityDocsURL,
			Marking: "blocking",
		}},
		ViolationCount: 1,
	}
}

var decisionID = func() domain.DecisionID {
	id, err := domain.ParseDecisionID(decisionText)
	if err != nil {
		panic(err)
	}
	return id
}()

func TestPublishCheckRunCreatesARunOnACommitThatHasNone(t *testing.T) {
	held := newCheckRunForge(t)
	run := securityRun()

	err := checkRunsAt(t, held.server).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, run)

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	held.mustHoldOnly(heldAs(1, run))
}

func TestPublishCheckRunPostsEveryFieldOfAViolationAndTheDecision(t *testing.T) {
	held := newCheckRunForge(t)

	err := checkRunsAt(t, held.server).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	published := held.onlyRun()
	const wireFailure = "failure"
	if published.Conclusion != wireFailure {
		t.Errorf("the conclusion is %q, want %q", published.Conclusion, wireFailure)
	}
	for field, want := range map[string]string{
		"rule id":       securityRuleID,
		"message":       securityMessage,
		"owning team":   securityTeam,
		"documentation": securityDocsURL,
	} {
		if !strings.Contains(published.Output.Text, want) {
			t.Errorf("the posted text does not carry the %s %q:\n%s", field, want, published.Output.Text)
		}
	}
	if !strings.Contains(published.Output.Summary, decisionText) {
		t.Errorf("the posted summary does not carry the decision %q: %q", decisionText, published.Output.Summary)
	}
}

func heldAs(id int64, run domain.CheckRun) wireCheckRun {
	return wireCheckRun{
		App:        wireApp{ID: forgeAppID},
		ID:         id,
		Name:       run.Name,
		HeadSHA:    headSHA,
		Status:     "completed",
		Conclusion: string(run.Conclusion),
		Output:     renderedOutputOf(run),
	}
}

func renderedOutputOf(run domain.CheckRun) wireOutput {
	title, summary, text, err := github.RenderedForGitHub(run)
	if err != nil {
		panic(err)
	}
	return wireOutput{Title: title, Summary: summary, Text: text}
}

func (held *checkRunForge) onlyRun() wireCheckRun {
	held.t.Helper()

	published := held.runsOn(headSHA)
	if len(published) != 1 {
		held.t.Fatalf("the forge holds %d runs, want 1", len(published))
	}
	return published[0]
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
	second.Headline = "every security policy admitted the change"
	second.Violations = nil
	second.ViolationCount = 0

	for _, run := range []domain.CheckRun{first, second} {
		if err := runs.PublishCheckRun(context.Background(), locator, headCommit, run); err != nil {
			t.Fatalf("publish check run: %v", err)
		}
	}

	held.mustHoldOnly(heldAs(1, second))
}

func TestPublishCheckRunNeverUpdatesARunAnotherAppOwns(t *testing.T) {
	held := newCheckRunForge(t)
	run := securityRun()
	foreign := heldAs(1, run)
	foreign.App = wireApp{ID: otherAppID}
	foreign.Conclusion = string(domain.ConclusionSuccess)
	held.runs = []wireCheckRun{foreign}

	err := checkRunsAt(t, held.server).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, run)

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	want := []wireCheckRun{foreign, heldAs(2, run)}
	if got := held.runsOn(headSHA); !slices.Equal(got, want) {
		t.Fatalf("the forge holds %+v, want the foreign run untouched beside ours: %+v", got, want)
	}
}

func TestPublishCheckRunChecksOwnershipEvenWhenTheForgeIgnoresTheAppFilter(t *testing.T) {
	held := newCheckRunForge(t)
	held.ignoresAppID = true
	run := securityRun()
	foreign := heldAs(1, run)
	foreign.App = wireApp{ID: otherAppID}
	foreign.Conclusion = string(domain.ConclusionSuccess)
	held.runs = []wireCheckRun{foreign}

	err := checkRunsAt(t, held.server).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, run)

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	want := []wireCheckRun{foreign, heldAs(2, run)}
	if got := held.runsOn(headSHA); !slices.Equal(got, want) {
		t.Fatalf("the forge holds %+v, want the foreign run untouched beside ours: %+v", got, want)
	}
}

func TestPublishCheckRunNeverUpdatesARunThatNamesNoApp(t *testing.T) {
	ownerless := []string{
		`{"total_count": 1, "check_runs": [{"id": 7, "app": null}]}`,
		`{"total_count": 1, "check_runs": [{"id": 7}]}`,
		`{"total_count": 1, "check_runs": [{"id": 7, "app": {}}]}`,
	}

	for _, listing := range ownerless {
		t.Run(listing, func(t *testing.T) {
			server := answeringWrite(t, failedStep{http.MethodPost, http.StatusCreated, listing}, `{}`)

			err := checkRunsAt(t, server).PublishCheckRun(
				context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

			if err != nil {
				t.Fatalf("publish check run: %v", err)
			}
		})
	}
}

func TestPublishCheckRunListsByNameOwnerAndLatest(t *testing.T) {
	mustListExactlyFor(t, securityRun().Name)
}

func TestPublishCheckRunListsANameWithQuerySyntaxAsOneValue(t *testing.T) {
	mustListExactlyFor(t, "policy/a&app_id=1 #frag 100%25 done")
}

func mustListExactlyFor(t *testing.T, name string) {
	t.Helper()

	var asked url.Values
	server := forge(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			asked = request.URL.Query()
			write(t, writer, noRunsListed)
			return
		}
		writer.WriteHeader(http.StatusCreated)
		write(t, writer, `{}`)
	})
	run := securityRun()
	run.Name = name

	err := checkRunsAt(t, server).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, run)

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	want := url.Values{
		"check_name": {name},
		"app_id":     {strconv.Itoa(forgeAppID)},
		"filter":     {"latest"},
		"per_page":   {strconv.Itoa(github.CheckRunsPerPage)},
	}
	if !maps.EqualFunc(asked, want, slices.Equal) {
		t.Fatalf("the listing asked %v, want %v", asked, want)
	}
}

func TestPublishCheckRunUpdatesTheNewestOfSeveralRunsOfTheSameName(t *testing.T) {
	held := newCheckRunForge(t)
	earlier := securityRun()
	held.runs = []wireCheckRun{heldAs(3, earlier), heldAs(7, earlier), heldAs(5, earlier)}
	later := earlier
	later.Conclusion = domain.ConclusionSuccess
	later.Title = "no violations"

	err := checkRunsAt(t, held.server).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, later)

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	want := []wireCheckRun{heldAs(3, earlier), heldAs(7, later), heldAs(5, earlier)}
	if got := held.runsOn(headSHA); !slices.Equal(got, want) {
		t.Fatalf("the forge holds %+v, want only the newest updated: %+v", got, want)
	}
}

func listingOnly(t *testing.T, status int, listing string) *httptest.Server {
	t.Helper()

	return forge(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("the adapter sent %s %s after the listing", request.Method, request.URL)
			writer.WriteHeader(http.StatusTeapot)
			return
		}
		writer.WriteHeader(status)
		write(t, writer, listing)
	})
}

func TestPublishCheckRunRefusesAListingThatDoesNotStateItsRuns(t *testing.T) {
	unstated := []string{
		`{}`,
		`null`,
		`{"message": "Bad credentials"}`,
		`{"total_count": 0}`,
		`{"check_runs": null}`,
		`{"total_count": 0, "check_runs": null}`,
		`{"total_count": null, "check_runs": []}`,
	}

	for _, listing := range unstated {
		t.Run(listing, func(t *testing.T) {
			runs := checkRunsAt(t, listingOnly(t, http.StatusOK, listing))

			err := runs.PublishCheckRun(context.Background(),
				locatorOf(t, "acme/payments", 412), headCommit, securityRun())

			if !errors.Is(err, github.ErrMalformedListing) {
				t.Fatalf("error is %v, want %v", err, github.ErrMalformedListing)
			}
		})
	}
}

func TestPublishCheckRunRefusesAListedRunWithoutAPositiveID(t *testing.T) {
	malformed := []string{
		`{"total_count": 1, "check_runs": [{}]}`,
		`{"total_count": 1, "check_runs": [{"id": 0}]}`,
		`{"total_count": 1, "check_runs": [{"id": -3}]}`,
		`{"total_count": 2, "check_runs": [{"id": 9}, {"id": 0}]}`,
	}

	for _, listing := range malformed {
		t.Run(listing, func(t *testing.T) {
			runs := checkRunsAt(t, listingOnly(t, http.StatusOK, listing))

			err := runs.PublishCheckRun(context.Background(),
				locatorOf(t, "acme/payments", 412), headCommit, securityRun())

			if !errors.Is(err, github.ErrMalformedListedRun) {
				t.Fatalf("error is %v, want %v", err, github.ErrMalformedListedRun)
			}
		})
	}
}

func TestPublishCheckRunRefusesAListingWhoseCountDisagreesWithItsRuns(t *testing.T) {
	disagreeing := []string{
		`{"total_count": 1, "check_runs": []}`,
		`{"total_count": 0, "check_runs": [{"id": 7}]}`,
	}

	for _, listing := range disagreeing {
		t.Run(listing, func(t *testing.T) {
			runs := checkRunsAt(t, listingOnly(t, http.StatusOK, listing))

			err := runs.PublishCheckRun(context.Background(),
				locatorOf(t, "acme/payments", 412), headCommit, securityRun())

			if !errors.Is(err, github.ErrInconsistentListing) {
				t.Fatalf("error is %v, want %v", err, github.ErrInconsistentListing)
			}
		})
	}
}

func TestPublishCheckRunRefusesAnUnpublishableRunWithoutAsking(t *testing.T) {
	const notUTF8 = "policy \xff refused"
	unpublishable := map[string]struct {
		spoil func(*domain.CheckRun)
		want  error
	}{
		"no name":            {func(run *domain.CheckRun) { run.Name = "" }, github.ErrIncompleteCheckRun},
		"no title":           {func(run *domain.CheckRun) { run.Title = "" }, github.ErrIncompleteCheckRun},
		"no headline":        {func(run *domain.CheckRun) { run.Headline = "" }, github.ErrIncompleteCheckRun},
		"no conclusion":      {func(run *domain.CheckRun) { run.Conclusion = "" }, domain.ErrUnknownConclusion},
		"skipped":            {func(run *domain.CheckRun) { run.Conclusion = "skipped" }, domain.ErrUnknownConclusion},
		"action_required":    {func(run *domain.CheckRun) { run.Conclusion = "action_required" }, domain.ErrUnknownConclusion},
		"SUCCESS":            {func(run *domain.CheckRun) { run.Conclusion = "SUCCESS" }, domain.ErrUnknownConclusion},
		"no decision":        {func(run *domain.CheckRun) { run.DecisionID = domain.DecisionID{} }, github.ErrUnrenderableCheckRun},
		"name not UTF-8":     {func(run *domain.CheckRun) { run.Name = notUTF8 }, github.ErrNotUTF8},
		"title not UTF-8":    {func(run *domain.CheckRun) { run.Title = notUTF8 }, github.ErrNotUTF8},
		"headline not UTF-8": {func(run *domain.CheckRun) { run.Headline = notUTF8 }, github.ErrNotUTF8},
	}
	runs := checkRunsAt(t, refusingEveryRequest(t))

	for name, refusal := range unpublishable {
		t.Run(name, func(t *testing.T) {
			run := securityRun()
			refusal.spoil(&run)

			err := runs.PublishCheckRun(
				context.Background(), locatorOf(t, "acme/payments", 412), headCommit, run)

			if !errors.Is(err, refusal.want) {
				t.Fatalf("error is %v, want %v", err, refusal.want)
			}
		})
	}
}

func TestPublishCheckRunFitsAViolationLongerThanTheForgeAcceptsWithinItsLimit(t *testing.T) {
	held := newCheckRunForge(t)
	run := securityRun()
	run.Violations[0].Message = strings.Repeat("é", github.MaxOutputCharacters+1)

	err := checkRunsAt(t, held.server).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, run)

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	published := held.onlyRun()
	for field, value := range map[string]string{
		"summary": published.Output.Summary, "text": published.Output.Text,
	} {
		if characters := utf8.RuneCountInString(value); characters > github.MaxOutputCharacters {
			t.Errorf("%s is %d characters, over the forge's %d", field, characters, github.MaxOutputCharacters)
		}
	}
	if !strings.Contains(published.Output.Text, "1 violation in total") {
		t.Errorf("the text does not count the violation it could not list: %q", published.Output.Text)
	}
}

func TestPublishCheckRunListsAViolationThatFitsInCharactersButNotInBytes(t *testing.T) {
	held := newCheckRunForge(t)
	run := securityRun()
	run.Violations[0].Message = strings.Repeat("é", 40000)

	err := checkRunsAt(t, held.server).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, run)

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	published := held.onlyRun()
	if !strings.Contains(published.Output.Text, run.Violations[0].Message) {
		t.Errorf("the text does not list a violation of %d characters", utf8.RuneCountInString(run.Violations[0].Message))
	}
}

func TestPublishCheckRunAcceptsASummaryOfExactlyTheForgesCharactersAndNotOneMore(t *testing.T) {
	held := newCheckRunForge(t)
	runs := checkRunsAt(t, held.server)
	locator := locatorOf(t, "acme/payments", 412)
	short := securityRun()
	if err := runs.PublishCheckRun(context.Background(), locator, headCommit, short); err != nil {
		t.Fatalf("publish a short headline: %v", err)
	}
	besideHeadline := utf8.RuneCountInString(held.onlyRun().Output.Summary) - utf8.RuneCountInString(short.Headline)
	exact := securityRun()
	exact.Headline = strings.Repeat("é", github.MaxOutputCharacters-besideHeadline)
	over := securityRun()
	over.Headline = exact.Headline + "é"

	if err := runs.PublishCheckRun(context.Background(), locator, headCommit, exact); err != nil {
		t.Errorf("a summary of exactly %d characters: %v", github.MaxOutputCharacters, err)
	}
	if got := utf8.RuneCountInString(held.onlyRun().Output.Summary); got != github.MaxOutputCharacters {
		t.Errorf("the forge holds a summary of %d characters, want %d", got, github.MaxOutputCharacters)
	}
	err := checkRunsAt(t, refusingEveryRequest(t)).PublishCheckRun(context.Background(), locator, headCommit, over)
	if !errors.Is(err, github.ErrUnrenderableCheckRun) {
		t.Errorf("a summary one character over: error is %v, want %v", err, github.ErrUnrenderableCheckRun)
	}
}

const (
	noRunsListed = `{"total_count": 0, "check_runs": []}`
	run7Listed   = `{"total_count": 1, "check_runs": [{"id": 7, "app": {"id": 4242}}]}`
)

type failedStep struct {
	method  string
	status  int
	listing string
}

func failingOn(t *testing.T, step failedStep) *httptest.Server {
	t.Helper()

	return forge(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == step.method {
			writer.WriteHeader(step.status)
			write(t, writer, `{"message": "`+request.Header.Get("Authorization")+`"}`)
			return
		}
		write(t, writer, step.listing)
	})
}

func TestPublishCheckRunAcceptsOnlyTheSuccessStatusEachCallPromises(t *testing.T) {
	otherSuccesses := map[string]failedStep{
		"listing created":          {http.MethodGet, http.StatusCreated, noRunsListed},
		"listing accepted":         {http.MethodGet, http.StatusAccepted, noRunsListed},
		"create ok":                {http.MethodPost, http.StatusOK, noRunsListed},
		"create non-authoritative": {http.MethodPost, http.StatusNonAuthoritativeInfo, noRunsListed},
		"update created":           {http.MethodPatch, http.StatusCreated, run7Listed},
		"update accepted":          {http.MethodPatch, http.StatusAccepted, run7Listed},
	}

	for name, step := range otherSuccesses {
		t.Run(name, func(t *testing.T) {
			err := checkRunsAt(t, failingOn(t, step)).PublishCheckRun(
				context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

			if !errors.Is(err, github.ErrUnexpectedStatus) {
				t.Fatalf("error is %v, want %v", err, github.ErrUnexpectedStatus)
			}
		})
	}
}

func TestPublishCheckRunNamesWhatTheForgeCouldNotFind(t *testing.T) {
	notFound := map[string]struct {
		step    failedStep
		missing string
	}{
		"listing": {failedStep{http.MethodGet, http.StatusNotFound, noRunsListed}, "commit " + headSHA},
		"create":  {failedStep{http.MethodPost, http.StatusNotFound, noRunsListed}, "repository acme/payments"},
		"update":  {failedStep{http.MethodPatch, http.StatusNotFound, run7Listed}, "check run 7"},
	}

	for step, answer := range notFound {
		t.Run(step, func(t *testing.T) {
			err := checkRunsAt(t, failingOn(t, answer.step)).PublishCheckRun(
				context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

			if !errors.Is(err, github.ErrCheckRunTargetNotFound) {
				t.Fatalf("error is %v, want %v", err, github.ErrCheckRunTargetNotFound)
			}
			if errors.Is(err, domain.ErrNoSuchPullRequest) {
				t.Errorf("error %v claims the pull request is missing", err)
			}
			if !strings.Contains(err.Error(), answer.missing) {
				t.Errorf("error %q does not name the missing %s", err, answer.missing)
			}
		})
	}
}

func TestPublishCheckRunReportsAFailedExchangeWithItsContextButNotItsCredential(t *testing.T) {
	failing := map[string]*httptest.Server{
		"listing": failingOn(t, failedStep{http.MethodGet, http.StatusUnprocessableEntity, noRunsListed}),
		"create":  failingOn(t, failedStep{http.MethodPost, http.StatusUnprocessableEntity, noRunsListed}),
		"update":  failingOn(t, failedStep{http.MethodPatch, http.StatusUnprocessableEntity, run7Listed}),
	}

	for step, server := range failing {
		t.Run(step, func(t *testing.T) {
			run := securityRun()

			err := checkRunsAt(t, server).PublishCheckRun(
				context.Background(), locatorOf(t, "acme/payments", 412), headCommit, run)

			if !errors.Is(err, github.ErrUnexpectedStatus) {
				t.Fatalf("error is %v, want %v", err, github.ErrUnexpectedStatus)
			}
			for _, context := range []string{run.Name, "acme/payments#412", headSHA} {
				if !strings.Contains(err.Error(), context) {
					t.Errorf("error %q does not name %q", err, context)
				}
			}
			if strings.Contains(err.Error(), forgeToken) {
				t.Errorf("error %q reveals the credential", err)
			}
		})
	}
}

func TestPublishCheckRunTreatsAWriteAnsweredWithALargeEchoAsDone(t *testing.T) {
	fitsOnlyTheListing := max(len(noRunsListed), len(run7Listed))
	echo := `{"output": {"text": "` + strings.Repeat("x", fitsOnlyTheListing) + `"}}`
	writes := map[string]failedStep{
		"create": {http.MethodPost, http.StatusCreated, noRunsListed},
		"update": {http.MethodPatch, http.StatusOK, run7Listed},
	}

	for name, write := range writes {
		t.Run(name, func(t *testing.T) {
			settings := settingsAt(answeringWrite(t, write, echo))
			settings.MaxResponseBytes = fitsOnlyTheListing

			err := checkRunsWith(t, settings).PublishCheckRun(
				context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

			if err != nil {
				t.Fatalf("publish check run: %v", err)
			}
		})
	}
}

func answeringWrite(t *testing.T, step failedStep, body string) *httptest.Server {
	t.Helper()

	return forge(t, func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			write(t, writer, step.listing)
		case step.method:
			writer.WriteHeader(step.status)
			write(t, writer, body)
		default:
			t.Errorf("the adapter sent %s %s, want %s", request.Method, request.URL, step.method)
			writer.WriteHeader(http.StatusTeapot)
		}
	})
}

type countedBody struct {
	unread     io.Reader
	read       int
	reachedEnd bool
}

func countedBodyOf(text string) *countedBody {
	return &countedBody{unread: strings.NewReader(text)}
}

func (body *countedBody) Read(into []byte) (int, error) {
	read, err := body.unread.Read(into)
	body.read += read
	body.reachedEnd = body.reachedEnd || errors.Is(err, io.EOF)
	return read, err
}

func (body *countedBody) Close() error { return nil }

type countedTransport struct{ listing, written *countedBody }

func (transport countedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Method != http.MethodGet {
		return &http.Response{
			StatusCode: http.StatusCreated, Status: "201 Created",
			Body: transport.written, Request: request,
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK, Status: "200 OK", Body: transport.listing, Request: request,
	}, nil
}

func publishThrough(t *testing.T, transport countedTransport, bound int) error {
	t.Helper()

	settings := wellFormedSettings()
	settings.Client = &http.Client{Transport: transport, Timeout: forgeTimeout}
	settings.MaxResponseBytes = bound
	return checkRunsWith(t, settings).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())
}

func publishOverListing(t *testing.T, listing string, bound int) (*countedBody, error) {
	t.Helper()

	body := countedBodyOf(listing)
	err := publishThrough(t, countedTransport{listing: body, written: countedBodyOf(`{}`)}, bound)
	return body, err
}

func TestPublishCheckRunDrainsAWriteBodyWithinTheBoundAndNoFurther(t *testing.T) {
	bound := len(noRunsListed)

	shortEcho := countedBodyOf(`{}`)
	err := publishThrough(t, countedTransport{listing: countedBodyOf(noRunsListed), written: shortEcho}, bound)
	if err != nil {
		t.Errorf("a short echo: %v", err)
	}
	if !shortEcho.reachedEnd {
		t.Errorf("a short echo was not read to its end, so the connection cannot be reused")
	}

	longEcho := countedBodyOf(strings.Repeat("x", 10*bound))
	err = publishThrough(t, countedTransport{listing: countedBodyOf(noRunsListed), written: longEcho}, bound)
	if err != nil {
		t.Errorf("an echo ten times the bound: %v", err)
	}
	if longEcho.read > bound+1 {
		t.Errorf("read %d bytes of a write echo bounded at %d", longEcho.read, bound)
	}
}

func TestPublishCheckRunIsHeldNoLongerThanTheClientTimeout(t *testing.T) {
	const (
		clientTimeout = 200 * time.Millisecond
		trickledBytes = 30
		trickleGap    = 100 * time.Millisecond
		heldAtMost    = time.Second
	)
	server := forge(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			write(t, writer, noRunsListed)
			return
		}
		writer.WriteHeader(http.StatusCreated)
		for range trickledBytes {
			select {
			case <-request.Context().Done():
				return
			case <-time.After(trickleGap):
			}
			if !isTrickled(writer) {
				return
			}
		}
	})
	settings := settingsAt(server)
	settings.Client.Timeout = clientTimeout

	started := time.Now()
	_ = checkRunsWith(t, settings).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, securityRun())

	if held := time.Since(started); held > heldAtMost {
		t.Fatalf("a trickling forge held the publish for %v, want at most %v", held, heldAtMost)
	}
}

func isTrickled(writer http.ResponseWriter) bool {
	if _, err := writer.Write([]byte(" ")); err != nil {
		return false
	}
	return http.NewResponseController(writer).Flush() == nil
}

func TestPublishCheckRunBoundsTheListingToExactlyTheResponseBound(t *testing.T) {
	exact := len(noRunsListed)

	if _, err := publishOverListing(t, noRunsListed, exact); err != nil {
		t.Errorf("a listing of exactly the bound: %v", err)
	}
	if _, err := publishOverListing(t, noRunsListed+" ", exact); !errors.Is(err, github.ErrResponseTooLarge) {
		t.Errorf("a listing one byte over the bound: error is %v, want %v", err, github.ErrResponseTooLarge)
	}
	body, err := publishOverListing(t, noRunsListed+strings.Repeat(" ", 10*exact), exact)
	if !errors.Is(err, github.ErrResponseTooLarge) {
		t.Errorf("a listing ten times the bound: error is %v, want %v", err, github.ErrResponseTooLarge)
	}
	if body.read > exact+1 {
		t.Errorf("read %d bytes of a listing bounded at %d", body.read, exact)
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

func TestPublishCheckRunRefusesAnUnparsedHeadWithoutAsking(t *testing.T) {
	runs := checkRunsAt(t, refusingEveryRequest(t))

	err := runs.PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), domain.CommitSHA{}, securityRun())

	if !errors.Is(err, github.ErrMalformedHeadSHA) {
		t.Fatalf("error is %v, want %v", err, github.ErrMalformedHeadSHA)
	}
}
