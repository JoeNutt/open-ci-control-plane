// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/extract/golang"
)

func TestGateChangePublishesOneCheckRunPerFamilyOnTheHeadCommit(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	gate, _ := recordingGate(t)
	published := &publishedCheckRuns{}
	id := decisionIDOf(t, gatedDecisionID)
	useCase := application.GateChange{
		Build:     buildOverAnAuthChange(t, locator),
		Gate:      gate,
		Publisher: published,
		Recorder:  &recordedDecisions{},
	}

	if err := useCase.Execute(context.Background(), locator, id); err != nil {
		t.Fatalf("gate change: %v", err)
	}

	runs := published.PublishedOn(locator, commitSHAOf(t, recordedHeadSHA))
	if len(runs) != len(domain.DeclaredPolicyFamilies()) {
		t.Fatalf("published %d check runs, want one per family: %v", len(runs), runs)
	}
	for _, family := range domain.DeclaredPolicyFamilies() {
		run, present := runs["policy/"+string(family)]
		if !present {
			t.Errorf("no check run for %s", family)
			continue
		}
		if run.DecisionID != id {
			t.Errorf("%s carries decision %q, want %q", family, run.DecisionID, id)
		}
	}
}

func TestGateChangePublishesACheckRunForEachConfiguredFamilyOnly(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	gate, err := domain.ParseGate(map[domain.PolicyFamily]domain.FamilyEngine{
		domain.PolicyFamilyStructural: &recordingEngine{revision: revisionForTest(t)},
	})
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}
	published := &publishedCheckRuns{}
	useCase := application.GateChange{
		Build:     buildOverAnAuthChange(t, locator),
		Gate:      gate,
		Publisher: published,
		Recorder:  &recordedDecisions{},
	}

	if err := useCase.Execute(context.Background(), locator, decisionIDOf(t, gatedDecisionID)); err != nil {
		t.Fatalf("gate change: %v", err)
	}

	runs := published.PublishedOn(locator, commitSHAOf(t, recordedHeadSHA))
	if _, present := runs["policy/structural"]; len(runs) != 1 || !present {
		t.Errorf("published %v, want the structural check run only", runs)
	}
}

func TestGateChangeRecordsEachFamilysDecisionUnderTheDecisionID(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	gate, _ := recordingGate(t)
	recorded := &recordedDecisions{}
	id := decisionIDOf(t, gatedDecisionID)
	useCase := application.GateChange{
		Build:     buildOverAnAuthChange(t, locator),
		Gate:      gate,
		Publisher: &publishedCheckRuns{},
		Recorder:  recorded,
	}

	if err := useCase.Execute(context.Background(), locator, id); err != nil {
		t.Fatalf("gate change: %v", err)
	}

	if len(recorded.records) != len(domain.DeclaredPolicyFamilies()) {
		t.Fatalf("recorded %d decisions, want one per family", len(recorded.records))
	}
	for _, record := range recorded.records {
		if record.DecisionID() != id {
			t.Errorf("recorded under %q, want %q", record.DecisionID(), id)
		}
	}
}

func TestARecorderFailureIsReturnedAndEveryCheckRunStillPublished(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	gate, _ := recordingGate(t)
	published := &publishedCheckRuns{}
	refusal := errors.New("decision sink unreachable")
	useCase := application.GateChange{
		Build:     buildOverAnAuthChange(t, locator),
		Gate:      gate,
		Publisher: published,
		Recorder:  refusingRecorder{refusal: refusal},
	}

	err := useCase.Execute(context.Background(), locator, decisionIDOf(t, gatedDecisionID))

	if !errors.Is(err, refusal) {
		t.Fatalf("gate change returned %v, want the recorder's failure", err)
	}
	runs := published.PublishedOn(locator, commitSHAOf(t, recordedHeadSHA))
	if len(runs) != len(domain.DeclaredPolicyFamilies()) {
		t.Errorf("published %d check runs, want one per family despite the recorder", len(runs))
	}
}

type refusingRecorder struct {
	refusal error
}

func (recorder refusingRecorder) RecordDecision(context.Context, domain.DecisionRecord) error {
	return recorder.refusal
}

type recordedDecisions struct {
	records []domain.DecisionRecord
}

func (recorded *recordedDecisions) RecordDecision(_ context.Context, record domain.DecisionRecord) error {
	recorded.records = append(recorded.records, record)
	return nil
}

func TestEveryFamilyDecidesOnlyTheDocumentBuiltFromTheForge(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	build := buildOverAnAuthChange(t, locator)
	gate, engines := recordingGate(t)
	useCase := application.GateChange{
		Build:     build,
		Gate:      gate,
		Publisher: &publishedCheckRuns{},
		Recorder:  &recordedDecisions{},
	}

	err := useCase.Execute(context.Background(), locator,
		decisionIDOf(t, gatedDecisionID))

	if err != nil {
		t.Fatalf("gate change: %v", err)
	}
	built, err := build.Execute(context.Background(), locator)
	if err != nil {
		t.Fatalf("build document: %v", err)
	}
	for family, engine := range engines {
		if len(engine.received) != 1 {
			t.Fatalf("%s was asked %d times, want once", family, len(engine.received))
		}
		if !reflect.DeepEqual(engine.received[0], built.Document) {
			t.Errorf("%s decided %+v, want the built document %+v",
				family, engine.received[0], built.Document)
		}
	}
}

func TestAChangeThatCannotBeBuiltIsNeverDecided(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 413)
	oversized := domain.DeclaredLimits().MaxPatchBytesPerFile + 1
	build := buildOverRecorded(t, locator, domain.File{
		Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
		Additions: 1, PatchBytes: oversized,
	})
	gate, engines := recordingGate(t)
	published := &publishedCheckRuns{}
	useCase := application.GateChange{
		Build:     build,
		Gate:      gate,
		Publisher: published,
		Recorder:  &recordedDecisions{},
	}

	err := useCase.Execute(context.Background(), locator,
		decisionIDOf(t, gatedDecisionID))

	if !errors.Is(err, domain.ErrFilePatchTooLarge) {
		t.Fatalf("want ErrFilePatchTooLarge, got %v", err)
	}
	for family, engine := range engines {
		if len(engine.received) != 0 {
			t.Errorf("%s was asked about a change that was never built", family)
		}
	}
}

func TestAFamilyWhoseCheckRunCannotBePostedDoesNotStopTheOthers(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	gate, _ := recordingGate(t)
	recorded := &publishedCheckRuns{}
	refused := errors.New("the forge refused the post")
	useCase := application.GateChange{
		Build:     buildOverAnAuthChange(t, locator),
		Gate:      gate,
		Publisher: refusingOne{name: "policy/structural", refusal: refused, rest: recorded},
		Recorder:  &recordedDecisions{},
	}

	err := useCase.Execute(context.Background(), locator,
		decisionIDOf(t, gatedDecisionID))

	if !errors.Is(err, refused) {
		t.Fatalf("want the refusal reported, got %v", err)
	}
	if !strings.Contains(err.Error(), "policy/structural") {
		t.Errorf("the refusal does not name the check run it refused: %v", err)
	}
	runs := recorded.PublishedOn(locator, commitSHAOf(t, recordedHeadSHA))
	if want := len(domain.DeclaredPolicyFamilies()) - 1; len(runs) != want {
		t.Errorf("published %d check runs, want the other %d: %v", len(runs), want, runs)
	}
}

type refusingOne struct {
	name    string
	refusal error
	rest    domain.CheckRunPublisher
}

func (publisher refusingOne) PublishCheckRun(
	ctx context.Context, locator domain.PullRequestLocator, head domain.CommitSHA, run domain.CheckRun,
) error {
	if run.Name == publisher.name {
		return publisher.refusal
	}
	return publisher.rest.PublishCheckRun(ctx, locator, head, run)
}

func TestADenyReachesItsFamilysCheckRunWithItsRuleOwnerLinkAndDecision(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	denied, err := domain.ParseDecision([]byte(`{
		"input_wellformed": true, "schema_version_accepted": 1, "effect": "block",
		"violations": [
			{ "rule_id": "AUTHZ-1", "message": "auth middleware changed with no authz test",
			  "severity": "deny", "owning_team": "security-platform",
			  "docs_url": "https://docs.example.com/policy/AUTHZ-1" }
		],
		"reviewers": [], "fail_closed": true
	}`), revisionForTest(t))
	if err != nil {
		t.Fatalf("parse decision: %v", err)
	}
	engines := map[domain.PolicyFamily]domain.FamilyEngine{}
	for _, family := range domain.DeclaredPolicyFamilies() {
		engines[family] = &recordingEngine{revision: revisionForTest(t)}
	}
	engines[domain.PolicyFamilySecurity] = answeringEngine{decided: denied}
	gate, err := domain.ParseGate(engines)
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}
	published := &publishedCheckRuns{}
	id := decisionIDOf(t, gatedDecisionID)
	useCase := application.GateChange{
		Build:     buildOverAnAuthChange(t, locator),
		Gate:      gate,
		Publisher: published,
		Recorder:  &recordedDecisions{},
	}

	if err := useCase.Execute(context.Background(), locator, id); err != nil {
		t.Fatalf("gate change: %v", err)
	}

	security := published.PublishedOn(locator, commitSHAOf(t, recordedHeadSHA))["policy/security"]
	if security.Conclusion != domain.ConclusionFailure {
		t.Errorf("a deny concluded %q, want %q", security.Conclusion, domain.ConclusionFailure)
	}
	if security.DecisionID != id {
		t.Errorf("the check run carries decision %q, want %q", security.DecisionID, id)
	}
	want := domain.CheckRunViolation{
		RuleID: "AUTHZ-1", Message: "auth middleware changed with no authz test",
		OwningTeam: "security-platform", DocsURL: "https://docs.example.com/policy/AUTHZ-1",
	}
	if len(security.Violations) != 1 || security.ViolationCount != 1 {
		t.Fatalf("listed %d of %d violations, want 1 of 1", len(security.Violations), security.ViolationCount)
	}
	listed := security.Violations[0]
	listed.Marking = ""
	if listed != want {
		t.Errorf("the check run lists %+v, want %+v", listed, want)
	}
}

type answeringEngine struct {
	decided domain.Decision
}

func (engine answeringEngine) Decide(context.Context, domain.Document) (domain.Decision, error) {
	return engine.decided, nil
}

func TestAChangeWhoseHeadCommitIsMalformedIsNeverPublished(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	build := buildOverAnAuthChange(t, locator)
	recorded := build.Source.(recordedChanges).Changes[locator]
	recorded.PullRequest.HeadSHA = "../../user/repos"
	build.Source.(recordedChanges).Changes[locator] = recorded
	gate, _ := recordingGate(t)
	published := &spyPublisher{}
	useCase := application.GateChange{
		Build:     build,
		Gate:      gate,
		Publisher: published,
		Recorder:  &recordedDecisions{},
	}

	err := useCase.Execute(context.Background(), locator, decisionIDOf(t, gatedDecisionID))

	if !errors.Is(err, domain.ErrMalformedCommitSHA) {
		t.Fatalf("want ErrMalformedCommitSHA, got %v", err)
	}
	if published.posts != 0 {
		t.Errorf("published %d check runs for a malformed head commit", published.posts)
	}
}

type spyPublisher struct {
	posts int
}

func (publisher *spyPublisher) PublishCheckRun(
	context.Context, domain.PullRequestLocator, domain.CommitSHA, domain.CheckRun,
) error {
	publisher.posts++
	return nil
}

func TestAVerdictReachedAtTheDeadlineIsStillPosted(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	engines := map[domain.PolicyFamily]domain.FamilyEngine{}
	for _, family := range domain.DeclaredPolicyFamilies() {
		engines[family] = stallingEngine{}
	}
	gate, err := domain.ParseGate(engines)
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}
	recorded := &publishedCheckRuns{}
	useCase := application.GateChange{
		Build:     buildOverAnAuthChange(t, locator),
		Gate:      gate,
		Publisher: honouringCancellation{rest: recorded},
		Recorder:  &recordedDecisions{},
	}
	deadline, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err = useCase.Execute(deadline, locator, decisionIDOf(t, gatedDecisionID))

	if err != nil {
		t.Fatalf("gate change: %v", err)
	}
	runs := recorded.PublishedOn(locator, commitSHAOf(t, recordedHeadSHA))
	if len(runs) != len(domain.DeclaredPolicyFamilies()) {
		t.Fatalf("posted %d check runs after the deadline, want one per family", len(runs))
	}
	for name, run := range runs {
		if run.Conclusion != domain.ConclusionFailure {
			t.Errorf("%s concluded %q at the deadline, want %q", name, run.Conclusion, domain.ConclusionFailure)
		}
	}
}

type stallingEngine struct{}

func (stallingEngine) Decide(ctx context.Context, _ domain.Document) (domain.Decision, error) {
	<-ctx.Done()
	return domain.Decision{}, ctx.Err()
}

type honouringCancellation struct {
	rest domain.CheckRunPublisher
}

func (publisher honouringCancellation) PublishCheckRun(
	ctx context.Context, locator domain.PullRequestLocator, head domain.CommitSHA, run domain.CheckRun,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return publisher.rest.PublishCheckRun(ctx, locator, head, run)
}

func decisionIDOf(t *testing.T, stated string) domain.DecisionID {
	t.Helper()

	id, err := domain.ParseDecisionID(stated)
	if err != nil {
		t.Fatalf("parse decision identifier: %v", err)
	}
	return id
}

func commitSHAOf(t *testing.T, stated string) domain.CommitSHA {
	t.Helper()

	head, err := domain.ParseCommitSHA(stated)
	if err != nil {
		t.Fatalf("parse commit SHA: %v", err)
	}
	return head
}

func recordingGate(t *testing.T) (domain.Gate, map[domain.PolicyFamily]*recordingEngine) {
	t.Helper()

	engines := map[domain.PolicyFamily]*recordingEngine{}
	held := map[domain.PolicyFamily]domain.FamilyEngine{}
	for _, family := range domain.DeclaredPolicyFamilies() {
		engines[family] = &recordingEngine{revision: revisionForTest(t)}
		held[family] = engines[family]
	}
	gate, err := domain.ParseGate(held)
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}
	return gate, engines
}

const gatedDecisionID = "dec_0123456789abcdef0123456789abcdef"

func buildOverAnAuthChange(t *testing.T, locator domain.PullRequestLocator) application.BuildDocument {
	t.Helper()

	return buildOverRecorded(t, locator, domain.File{
		Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
		Additions: 12, Deletions: 3, PatchBytes: 1840,
	})
}

const recordedHeadSHA = "9f1cabc0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6"

func buildOverRecorded(
	t *testing.T, locator domain.PullRequestLocator, file domain.File,
) application.BuildDocument {
	t.Helper()

	return application.BuildDocument{
		Source: recordedChanges{
			Changes: map[domain.PullRequestLocator]recordedChange{
				locator: {
					PullRequest: domain.PullRequest{
						Number: locator.Number(), Repository: locator.Repository(),
						HeadSHA: recordedHeadSHA},
					Tips:  recordedTips(t),
					Files: []domain.File{file},
				},
			},
		},
		Bases: recordedBases(t),
		Contents: recordedAtBothEnds(t, locator, file.Path,
			sourceImportingContext, sourceImportingContext),
		Extractor: golang.Extractor{},
		Tests:     golang.Extractor{},
		Limits:    domain.DeclaredLimits(),
	}
}

const sourceImportingContext = "package auth\n\nimport \"context\"\n"

const recordedBaseSHA = "0a1b2c3d4e5f60718293a4b5c6d7e8f901234567"

const recordedBaseTipSHA = "7c6b5a4938271605f4e3d2c1b0a9f8e7d6c5b4a3"

func recordedTips(t *testing.T) domain.BranchTips {
	t.Helper()

	tips, err := domain.ParseBranchTips(commitSHAOf(t, recordedBaseTipSHA), commitSHAOf(t, recordedHeadSHA))
	if err != nil {
		t.Fatalf("parse branch tips: %v", err)
	}
	return tips
}

func recordedBases(t *testing.T) recordedMergeBases {
	t.Helper()

	return recordedMergeBases{MergeBases: map[domain.BranchTips]domain.CommitSHA{
		recordedTips(t): commitSHAOf(t, recordedBaseSHA),
	}}
}

func recordedAtBothEnds(
	t *testing.T, locator domain.PullRequestLocator, path, baseSource, headSource string,
) recordedFiles {
	t.Helper()

	contents := recordedAt(t, locator, recordedHeadSHA, path, headSource)
	maps.Copy(contents.Files, recordedAt(t, locator, recordedBaseSHA, path, baseSource).Files)
	return contents
}

func recordedAt(
	t *testing.T, locator domain.PullRequestLocator, commit, path, source string,
) recordedFiles {
	t.Helper()

	file, err := domain.ParseFileAtCommit(locator.GovernedRepository(), commitSHAOf(t, commit), path)
	if err != nil {
		t.Fatalf("parse file at commit: %v", err)
	}
	return recordedFiles{Files: map[domain.FileAtCommit][]byte{file: []byte(source)}}
}

type recordingEngine struct {
	revision domain.BundleRevision
	received []domain.Document
}

func (engine *recordingEngine) Decide(
	_ context.Context, document domain.Document,
) (domain.Decision, error) {
	engine.received = append(engine.received, document)
	return domain.UndecidedDecision(engine.revision), nil
}

func revisionForTest(t *testing.T) domain.BundleRevision {
	t.Helper()
	revision, err := domain.ParseBundleRevision("a11ce0ffee0000000000000000000000000000a1")
	if err != nil {
		t.Fatalf("parse revision: %v", err)
	}
	return revision
}
