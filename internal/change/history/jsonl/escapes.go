// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var (
	_ domain.EscapeLog     = (*Escapes)(nil)
	_ domain.EscapeHistory = (*Escapes)(nil)
)

type Escapes struct {
	log *Log
}

func NewEscapes(log *Log) *Escapes {
	return &Escapes{log: log}
}

type escapeLine struct {
	Kind         lineKind `json:"kind"`
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
		Kind: escapeKind, Mode: string(escape.Mode), Repository: escape.PullRequest.Repository(), PullRequest: escape.PullRequest.Number(),
		Suite: escape.Test.Suite.String(), Test: escape.Test.Name, Outcome: string(escape.Test.Outcome), Rerun: string(escape.Test.Rerun),
		ChangedFiles: namesOf(escape.ChangedFiles),
	})
	if err != nil {
		return fmt.Errorf("encode the %s escape of %s: %w", escape.Mode, escape.Test.Name, err)
	}
	return escapes.log.Append(body)
}

func (escapes *Escapes) RecordedEscapes(_ context.Context) ([]domain.Escape, error) {
	entries, err := escapes.log.Entries()
	if err != nil {
		return nil, fmt.Errorf("read the shadow history: %w", err)
	}
	var recorded []domain.Escape
	for number, entry := range entries {
		var line escapeLine
		if err := json.Unmarshal(entry.Body, &line); err != nil {
			return nil, fmt.Errorf("read shadow history entry %d: %w", number+1, err)
		}
		if line.Kind != escapeKind {
			continue
		}
		escape, err := line.escape()
		if err != nil {
			return nil, fmt.Errorf("read shadow history entry %d: %w", number+1, err)
		}
		recorded = append(recorded, escape)
	}
	return recorded, nil
}

func (line escapeLine) escape() (domain.Escape, error) {
	mode, err := domain.ParseShadowedMode(line.Mode)
	if err != nil {
		return domain.Escape{}, err
	}
	locator, err := domain.ParsePullRequestLocator(line.Repository, line.PullRequest)
	if err != nil {
		return domain.Escape{}, err
	}
	test, err := domain.ParseTestResult(domain.TestResultStatement{Suite: line.Suite, Name: line.Test, Outcome: line.Outcome, Rerun: line.Rerun})
	if err != nil {
		return domain.Escape{}, err
	}
	changed, err := parsedAll(line.ChangedFiles, domain.ParseRepositoryPath)
	if err != nil {
		return domain.Escape{}, err
	}
	return domain.Escape{Mode: mode, PullRequest: locator, Test: test, ChangedFiles: changed}, nil
}
