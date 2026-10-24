// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var (
	_ domain.PlanLedger = (*PlanLedger)(nil)
	_ domain.ShadowRuns = (*PlanLedger)(nil)
)

type PlanLedger struct {
	reporting sync.Mutex
	log       *Log
}

func NewPlanLedger(log *Log) *PlanLedger {
	return &PlanLedger{log: log}
}

func (ledger *PlanLedger) RecordDispatch(_ context.Context, dispatched domain.DispatchedPlan) error {
	if err := ledger.append(dispatchedLineOf(dispatched)); err != nil {
		return fmt.Errorf("record the plan dispatched for %s: %w", dispatched.Head, err)
	}
	return nil
}

func (ledger *PlanLedger) DispatchedPlanFor(
	_ context.Context, repository domain.GovernedRepository, head domain.CommitSHA,
) (domain.DispatchedPlan, error) {
	line, isFound, err := ledger.latestLineFor(ledgerKey{kind: dispatchedKind, repository: repository, head: head})
	if err != nil {
		return domain.DispatchedPlan{}, err
	}
	if !isFound {
		return domain.DispatchedPlan{}, domain.ErrNoPlanForHead
	}
	return line.dispatchedPlan(head)
}

func (ledger *PlanLedger) RecordReport(_ context.Context, repository domain.GovernedRepository, head domain.CommitSHA) error {
	ledger.reporting.Lock()
	defer ledger.reporting.Unlock()

	_, isReported, err := ledger.latestLineFor(ledgerKey{kind: reportedKind, repository: repository, head: head})
	if err != nil {
		return err
	}
	if isReported {
		return domain.ErrRunAlreadyReported
	}
	if err := ledger.append(ledgerLine{Kind: reportedKind, Repository: repository.String(), Head: head.String()}); err != nil {
		return fmt.Errorf("record the report for %s: %w", head, err)
	}
	return nil
}

type lineKind string

const (
	dispatchedKind lineKind = "dispatched"
	reportedKind   lineKind = "reported"
	escapeKind     lineKind = "escape"
)

type ledgerKey struct {
	kind       lineKind
	repository domain.GovernedRepository
	head       domain.CommitSHA
}

func (key ledgerKey) isHeldBy(line ledgerLine) bool {
	return line.Kind == key.kind && line.Repository == key.repository.String() && line.Head == key.head.String()
}

func (ledger *PlanLedger) latestLineFor(key ledgerKey) (ledgerLine, bool, error) {
	entries, err := ledger.log.Entries()
	if err != nil {
		return ledgerLine{}, false, fmt.Errorf("read the plan ledger: %w", err)
	}
	var latest ledgerLine
	isFound := false
	for number, entry := range entries {
		var line ledgerLine
		if err := json.Unmarshal(entry.Body, &line); err != nil {
			return ledgerLine{}, false, fmt.Errorf("read plan ledger entry %d: %w", number+1, err)
		}
		if key.isHeldBy(line) {
			latest, isFound = line, true
		}
	}
	return latest, isFound, nil
}

func (ledger *PlanLedger) append(line ledgerLine) error {
	body, err := json.Marshal(line)
	if err != nil {
		return fmt.Errorf("encode a plan ledger line: %w", err)
	}
	return ledger.log.Append(body)
}

type ledgerLine struct {
	Kind         lineKind      `json:"kind"`
	Repository   string        `json:"repository"`
	PullRequest  int           `json:"pullRequest,omitempty"`
	Head         string        `json:"head"`
	DecisionID   string        `json:"decisionId,omitempty"`
	Suites       []string      `json:"suites,omitempty"`
	AlwaysRun    []string      `json:"alwaysRun,omitempty"`
	Recorded     *recordedLine `json:"recorded,omitempty"`
	ChangedFiles []string      `json:"changedFiles,omitempty"`
	Modes        *modesLine    `json:"modes,omitempty"`
}

type modesLine struct {
	Selection    string `json:"selection"`
	Minimisation string `json:"minimisation"`
}

type recordedLine struct {
	Selected  []packageTestsLine `json:"selected"`
	Minimised []packageTestsLine `json:"minimised"`
}

type packageTestsLine struct {
	Package string   `json:"package"`
	Names   []string `json:"names"`
}

func dispatchedLineOf(dispatched domain.DispatchedPlan) ledgerLine {
	return ledgerLine{
		Kind:         dispatchedKind,
		Repository:   dispatched.Locator.Repository(),
		PullRequest:  dispatched.Locator.Number(),
		Head:         dispatched.Head.String(),
		DecisionID:   dispatched.DecisionID.String(),
		Suites:       namesOf(dispatched.Suites),
		AlwaysRun:    namesOf(dispatched.AlwaysRun),
		Recorded:     recordedLineOf(dispatched.Recorded),
		ChangedFiles: namesOf(dispatched.ChangedFiles),
		Modes:        modesLineOf(dispatched.Modes),
	}
}

func recordedLineOf(recorded domain.RecordedSets) *recordedLine {
	if !recorded.IsRecorded {
		return nil
	}
	return &recordedLine{Selected: packageTestsLinesOf(recorded.Selected), Minimised: packageTestsLinesOf(recorded.Minimised)}
}

func modesLineOf(modes domain.PlanModes) *modesLine {
	if modes == (domain.PlanModes{}) {
		return nil
	}
	return &modesLine{Selection: modes.Selection.String(), Minimisation: modes.Minimisation.String()}
}

func namesOf[Named fmt.Stringer](values []Named) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		names = append(names, value.String())
	}
	return names
}

func (line ledgerLine) dispatchedPlan(head domain.CommitSHA) (domain.DispatchedPlan, error) {
	dispatched, err := line.parsedPlan(head)
	if err != nil {
		return domain.DispatchedPlan{}, fmt.Errorf("read the plan dispatched for %s: %w", head, err)
	}
	return dispatched, nil
}

func (line ledgerLine) parsedPlan(head domain.CommitSHA) (domain.DispatchedPlan, error) {
	locator, err := domain.ParsePullRequestLocator(line.Repository, line.PullRequest)
	if err != nil {
		return domain.DispatchedPlan{}, err
	}
	id, err := domain.ParseDecisionID(line.DecisionID)
	if err != nil {
		return domain.DispatchedPlan{}, err
	}
	suites, err := parsedAll(line.Suites, domain.ParseCoverageSuite)
	if err != nil {
		return domain.DispatchedPlan{}, err
	}
	alwaysRun, err := parsedAll(line.AlwaysRun, domain.ParseGoImportPath)
	if err != nil {
		return domain.DispatchedPlan{}, err
	}
	changed, err := parsedAll(line.ChangedFiles, domain.ParseRepositoryPath)
	if err != nil {
		return domain.DispatchedPlan{}, err
	}
	recorded, err := line.Recorded.recordedSets()
	if err != nil {
		return domain.DispatchedPlan{}, err
	}
	modes, err := line.Modes.planModes()
	if err != nil {
		return domain.DispatchedPlan{}, err
	}
	return domain.DispatchedPlan{
		Locator: locator, Head: head, DecisionID: id, Suites: suites,
		AlwaysRun: alwaysRun, Recorded: recorded, ChangedFiles: changed, Modes: modes,
	}, nil
}

func (line *recordedLine) recordedSets() (domain.RecordedSets, error) {
	if line == nil {
		return domain.RecordedSets{}, nil
	}
	selected, err := parsedAll(line.Selected, packageTestsLine.packageTests)
	if err != nil {
		return domain.RecordedSets{}, err
	}
	minimised, err := parsedAll(line.Minimised, packageTestsLine.packageTests)
	if err != nil {
		return domain.RecordedSets{}, err
	}
	return domain.RecordedSets{IsRecorded: true, Selected: selected, Minimised: minimised}, nil
}

func (line *packageTestsLine) UnmarshalJSON(stated []byte) error {
	if isStatedAsSuite(stated) {
		line.Names = nil
		return json.Unmarshal(stated, &line.Package)
	}
	type objectLine packageTestsLine
	var held objectLine
	if err := json.Unmarshal(stated, &held); err != nil {
		return err
	}
	*line = packageTestsLine(held)
	return nil
}

func isStatedAsSuite(stated []byte) bool {
	trimmed := bytes.TrimSpace(stated)
	return len(trimmed) > 0 && trimmed[0] == '"'
}

func (line packageTestsLine) packageTests() (domain.PackageTests, error) {
	pkg, err := domain.ParseGoImportPath(line.Package)
	if err != nil {
		return domain.PackageTests{}, err
	}
	names, err := parsedAll(line.Names, domain.ParseTopLevelTestName)
	if err != nil {
		return domain.PackageTests{}, err
	}
	return domain.NewPackageTests(pkg, names), nil
}

func packageTestsLinesOf(tests []domain.PackageTests) []packageTestsLine {
	lines := make([]packageTestsLine, 0, len(tests))
	for _, entry := range tests {
		lines = append(lines, packageTestsLine{Package: entry.Package().String(), Names: namesOf(entry.Names())})
	}
	return lines
}

func (line *modesLine) planModes() (domain.PlanModes, error) {
	if line == nil {
		return domain.PlanModes{}, nil
	}
	selection, err := domain.ParseTestMode(line.Selection)
	if err != nil {
		return domain.PlanModes{}, err
	}
	minimisation, err := domain.ParseTestMode(line.Minimisation)
	if err != nil {
		return domain.PlanModes{}, err
	}
	return domain.PlanModes{Selection: selection, Minimisation: minimisation}, nil
}

func parsedAll[Stated, Parsed any](stated []Stated, parse func(Stated) (Parsed, error)) ([]Parsed, error) {
	parsed := make([]Parsed, 0, len(stated))
	for _, value := range stated {
		one, err := parse(value)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, one)
	}
	return parsed, nil
}

func (ledger *PlanLedger) ShadowedRuns(_ context.Context) ([]domain.ShadowedRun, error) {
	entries, err := ledger.log.Entries()
	if err != nil {
		return nil, fmt.Errorf("read the plan ledger: %w", err)
	}
	held := newHeldRuns()
	for number, entry := range entries {
		var line ledgerLine
		if err := json.Unmarshal(entry.Body, &line); err != nil {
			return nil, fmt.Errorf("read plan ledger entry %d: %w", number+1, err)
		}
		held.hold(line, entry.At)
	}
	return held.runs()
}

type runKey struct {
	repository string
	head       string
}

type heldDispatch struct {
	line ledgerLine
	at   time.Time
}

type heldRuns struct {
	latest     map[runKey]heldDispatch
	order      []runKey
	isReported map[runKey]bool
}

func newHeldRuns() *heldRuns {
	return &heldRuns{latest: map[runKey]heldDispatch{}, isReported: map[runKey]bool{}}
}

func (held *heldRuns) hold(line ledgerLine, at time.Time) {
	key := runKey{repository: line.Repository, head: line.Head}
	switch line.Kind {
	case dispatchedKind:
		if _, isSeen := held.latest[key]; !isSeen {
			held.order = append(held.order, key)
		}
		held.latest[key] = heldDispatch{line: line, at: at}
	case reportedKind:
		held.isReported[key] = true
	}
}

func (held *heldRuns) runs() ([]domain.ShadowedRun, error) {
	runs := make([]domain.ShadowedRun, 0, len(held.order))
	for _, key := range held.order {
		head, err := domain.ParseCommitSHA(key.head)
		if err != nil {
			return nil, fmt.Errorf("read the plan ledger: %w", err)
		}
		latest := held.latest[key]
		plan, err := latest.line.dispatchedPlan(head)
		if err != nil {
			return nil, err
		}
		runs = append(runs, domain.ShadowedRun{At: latest.at, Plan: plan, IsReported: held.isReported[key]})
	}
	return runs, nil
}
