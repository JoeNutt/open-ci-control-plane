// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl

import (
	"context"
	"encoding/json"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type PlanLedger struct {
	log *Log
}

func NewPlanLedger(log *Log) *PlanLedger {
	return &PlanLedger{log: log}
}

type dispatchedLine struct {
	Repository  string   `json:"repository"`
	PullRequest int      `json:"pullRequest"`
	Head        string   `json:"head"`
	DecisionID  string   `json:"decisionId"`
	Suites      []string `json:"suites"`
}

func (ledger *PlanLedger) RecordDispatch(_ context.Context, dispatched domain.DispatchedPlan) error {
	var suites []string
	for _, suite := range dispatched.Suites {
		suites = append(suites, suite.String())
	}
	body, err := json.Marshal(dispatchedLine{
		Repository:  dispatched.Locator.Repository(),
		PullRequest: dispatched.Locator.Number(),
		Head:        dispatched.Head.String(),
		DecisionID:  dispatched.DecisionID.String(),
		Suites:      suites,
	})
	if err != nil {
		return err
	}
	return ledger.log.Append(body)
}

func (ledger *PlanLedger) DispatchedPlanFor(
	_ context.Context, repository domain.GovernedRepository, head domain.CommitSHA,
) (domain.DispatchedPlan, error) {
	entries, err := ledger.log.Entries()
	if err != nil {
		return domain.DispatchedPlan{}, err
	}
	for _, entry := range entries {
		var line dispatchedLine
		if err := json.Unmarshal(entry.Body, &line); err != nil {
			return domain.DispatchedPlan{}, err
		}
		if line.Repository != repository.String() || line.Head != head.String() {
			continue
		}
		locator, err := domain.ParsePullRequestLocator(line.Repository, line.PullRequest)
		if err != nil {
			return domain.DispatchedPlan{}, err
		}
		id, err := domain.ParseDecisionID(line.DecisionID)
		if err != nil {
			return domain.DispatchedPlan{}, err
		}
		var suites []domain.CoverageSuite
		for _, stated := range line.Suites {
			suite, err := domain.ParseCoverageSuite(stated)
			if err != nil {
				return domain.DispatchedPlan{}, err
			}
			suites = append(suites, suite)
		}
		return domain.DispatchedPlan{Locator: locator, Head: head, DecisionID: id, Suites: suites}, nil
	}
	return domain.DispatchedPlan{}, domain.ErrNoPlanForHead
}
