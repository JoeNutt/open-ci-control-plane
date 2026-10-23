// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
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
	line, isFound, err := ledger.lineFor(ledgerKey{kind: dispatchedKind, repository: repository, head: head})
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

	_, isReported, err := ledger.lineFor(ledgerKey{kind: reportedKind, repository: repository, head: head})
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

func (ledger *PlanLedger) lineFor(key ledgerKey) (ledgerLine, bool, error) {
	entries, err := ledger.log.Entries()
	if err != nil {
		return ledgerLine{}, false, fmt.Errorf("read the plan ledger: %w", err)
	}
	for number, entry := range entries {
		var line ledgerLine
		if err := json.Unmarshal(entry.Body, &line); err != nil {
			return ledgerLine{}, false, fmt.Errorf("read plan ledger entry %d: %w", number+1, err)
		}
		if key.isHeldBy(line) {
			return line, true, nil
		}
	}
	return ledgerLine{}, false, nil
}

func (ledger *PlanLedger) append(line ledgerLine) error {
	body, err := json.Marshal(line)
	if err != nil {
		return fmt.Errorf("encode a plan ledger line: %w", err)
	}
	return ledger.log.Append(body)
}

type ledgerLine struct {
	Kind        lineKind `json:"kind"`
	Repository  string   `json:"repository"`
	PullRequest int      `json:"pullRequest,omitempty"`
	Head        string   `json:"head"`
	DecisionID  string   `json:"decisionId,omitempty"`
	Suites      []string `json:"suites,omitempty"`
}

func dispatchedLineOf(dispatched domain.DispatchedPlan) ledgerLine {
	suites := make([]string, 0, len(dispatched.Suites))
	for _, suite := range dispatched.Suites {
		suites = append(suites, suite.String())
	}
	return ledgerLine{
		Kind:        dispatchedKind,
		Repository:  dispatched.Locator.Repository(),
		PullRequest: dispatched.Locator.Number(),
		Head:        dispatched.Head.String(),
		DecisionID:  dispatched.DecisionID.String(),
		Suites:      suites,
	}
}

func (line ledgerLine) dispatchedPlan(head domain.CommitSHA) (domain.DispatchedPlan, error) {
	locator, err := domain.ParsePullRequestLocator(line.Repository, line.PullRequest)
	if err != nil {
		return domain.DispatchedPlan{}, fmt.Errorf("read the plan dispatched for %s: %w", head, err)
	}
	id, err := domain.ParseDecisionID(line.DecisionID)
	if err != nil {
		return domain.DispatchedPlan{}, fmt.Errorf("read the plan dispatched for %s: %w", head, err)
	}
	suites, err := suitesOf(line.Suites)
	if err != nil {
		return domain.DispatchedPlan{}, fmt.Errorf("read the plan dispatched for %s: %w", head, err)
	}
	return domain.DispatchedPlan{Locator: locator, Head: head, DecisionID: id, Suites: suites}, nil
}

func suitesOf(stated []string) ([]domain.CoverageSuite, error) {
	suites := make([]domain.CoverageSuite, 0, len(stated))
	for _, name := range stated {
		suite, err := domain.ParseCoverageSuite(name)
		if err != nil {
			return nil, err
		}
		suites = append(suites, suite)
	}
	return suites, nil
}
