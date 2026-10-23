// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type PlanLedger struct {
	log *Log
}

func NewPlanLedger(log *Log) *PlanLedger {
	return &PlanLedger{log: log}
}

func (ledger *PlanLedger) RecordDispatch(_ context.Context, dispatched domain.DispatchedPlan) error {
	body, err := json.Marshal(dispatchedLineOf(dispatched))
	if err != nil {
		return fmt.Errorf("encode the plan dispatched for %s: %w", dispatched.Head, err)
	}
	return ledger.log.Append(body)
}

func (ledger *PlanLedger) DispatchedPlanFor(
	_ context.Context, repository domain.GovernedRepository, head domain.CommitSHA,
) (domain.DispatchedPlan, error) {
	entries, err := ledger.log.Entries()
	if err != nil {
		return domain.DispatchedPlan{}, fmt.Errorf("read the plan ledger: %w", err)
	}
	for number, entry := range entries {
		var line dispatchedLine
		if err := json.Unmarshal(entry.Body, &line); err != nil {
			return domain.DispatchedPlan{}, fmt.Errorf("read plan ledger entry %d: %w", number+1, err)
		}
		if line.Kind == "dispatched" && line.Repository == repository.String() && line.Head == head.String() {
			return line.dispatchedPlan(head)
		}
	}
	return domain.DispatchedPlan{}, domain.ErrNoPlanForHead
}

func (ledger *PlanLedger) RecordReport(_ context.Context, repository domain.GovernedRepository, head domain.CommitSHA) error {
	entries, err := ledger.log.Entries()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		var line dispatchedLine
		if err := json.Unmarshal(entry.Body, &line); err != nil {
			return err
		}
		if line.Kind == "reported" && line.Repository == repository.String() && line.Head == head.String() {
			return domain.ErrRunAlreadyReported
		}
	}
	body, err := json.Marshal(dispatchedLine{Kind: "reported", Repository: repository.String(), Head: head.String()})
	if err != nil {
		return err
	}
	return ledger.log.Append(body)
}

type dispatchedLine struct {
	Kind        string   `json:"kind"`
	Repository  string   `json:"repository"`
	PullRequest int      `json:"pullRequest"`
	Head        string   `json:"head"`
	DecisionID  string   `json:"decisionId"`
	Suites      []string `json:"suites"`
}

func dispatchedLineOf(dispatched domain.DispatchedPlan) dispatchedLine {
	suites := make([]string, 0, len(dispatched.Suites))
	for _, suite := range dispatched.Suites {
		suites = append(suites, suite.String())
	}
	return dispatchedLine{
		Kind:        "dispatched",
		Repository:  dispatched.Locator.Repository(),
		PullRequest: dispatched.Locator.Number(),
		Head:        dispatched.Head.String(),
		DecisionID:  dispatched.DecisionID.String(),
		Suites:      suites,
	}
}

func (line dispatchedLine) dispatchedPlan(head domain.CommitSHA) (domain.DispatchedPlan, error) {
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
