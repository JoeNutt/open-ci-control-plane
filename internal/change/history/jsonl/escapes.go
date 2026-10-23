// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl

import (
	"context"
	"encoding/json"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type Escapes struct {
	log *Log
}

func NewEscapes(log *Log) *Escapes {
	return &Escapes{log: log}
}

type escapeLine struct {
	Kind         string   `json:"kind"`
	Mode         string   `json:"mode"`
	Repository   string   `json:"repository"`
	PullRequest  int      `json:"pullRequest"`
	Suite        string   `json:"suite"`
	Test         string   `json:"test"`
	Outcome      string   `json:"outcome"`
	Rerun        string   `json:"rerun"`
	ChangedFiles []string `json:"changedFiles"`
}

func (escapes *Escapes) RecordEscape(_ context.Context, escape domain.Escape) error {
	body, err := json.Marshal(escapeLine{
		Kind: "escape", Mode: string(escape.Mode), Repository: escape.PullRequest.Repository(), PullRequest: escape.PullRequest.Number(),
		Suite: escape.Test.Suite.String(), Test: escape.Test.Name, Outcome: string(escape.Test.Outcome), Rerun: string(escape.Test.Rerun),
		ChangedFiles: namesOf(escape.ChangedFiles),
	})
	if err != nil {
		return err
	}
	return escapes.log.Append(body)
}

func (escapes *Escapes) RecordedEscapes() ([]domain.Escape, error) {
	entries, err := escapes.log.Entries()
	if err != nil {
		return nil, err
	}
	var recorded []domain.Escape
	for _, entry := range entries {
		var line escapeLine
		if err := json.Unmarshal(entry.Body, &line); err != nil {
			return nil, err
		}
		if line.Kind != "escape" {
			continue
		}
		locator, err := domain.ParsePullRequestLocator(line.Repository, line.PullRequest)
		if err != nil {
			return nil, err
		}
		test, err := domain.ParseTestResult(domain.TestResultStatement{Suite: line.Suite, Name: line.Test, Outcome: line.Outcome, Rerun: line.Rerun})
		if err != nil {
			return nil, err
		}
		changed, err := parsedAll(line.ChangedFiles, domain.ParseRepositoryPath)
		if err != nil {
			return nil, err
		}
		recorded = append(recorded, domain.Escape{Mode: domain.ShadowedMode(line.Mode), PullRequest: locator, Test: test, ChangedFiles: changed})
	}
	return recorded, nil
}
