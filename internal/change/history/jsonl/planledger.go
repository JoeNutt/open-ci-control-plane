// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var _ domain.PlanLedger = (*PlanLedger)(nil)

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
}

type recordedLine struct {
	Selected  []string `json:"selected"`
	Minimised []string `json:"minimised"`
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
	}
}

func recordedLineOf(recorded domain.RecordedSets) *recordedLine {
	if !recorded.IsRecorded {
		return nil
	}
	return &recordedLine{Selected: namesOf(recorded.Selected), Minimised: namesOf(recorded.Minimised)}
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
	return domain.DispatchedPlan{
		Locator: locator, Head: head, DecisionID: id, Suites: suites,
		AlwaysRun: alwaysRun, Recorded: recorded, ChangedFiles: changed,
	}, nil
}

func (line *recordedLine) recordedSets() (domain.RecordedSets, error) {
	if line == nil {
		return domain.RecordedSets{}, nil
	}
	selected, err := parsedAll(line.Selected, domain.ParseCoverageSuite)
	if err != nil {
		return domain.RecordedSets{}, err
	}
	minimised, err := parsedAll(line.Minimised, domain.ParseCoverageSuite)
	if err != nil {
		return domain.RecordedSets{}, err
	}
	return domain.RecordedSets{IsRecorded: true, Selected: selected, Minimised: minimised}, nil
}

func parsedAll[Parsed any](stated []string, parse func(string) (Parsed, error)) ([]Parsed, error) {
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
