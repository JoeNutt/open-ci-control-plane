// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"errors"
	"fmt"
)

var ErrMalformedDecision = errors.New("decision document is malformed")

type Violation struct {
	RuleID     string              `json:"rule_id"`
	Message    string              `json:"message"`
	Severity   EnforcementSeverity `json:"severity"`
	OwningTeam string              `json:"owning_team"`
	DocsURL    string              `json:"docs_url"`
}

type Decision struct {
	outcome    Outcome
	effect     Effect
	violations Declared[Violation]
	reviewers  Declared[string]
	failClosed bool
	posture    FailurePosture
}

func (decided Decision) Outcome() Outcome                { return decided.outcome }
func (decided Decision) Effect() Effect                  { return decided.effect }
func (decided Decision) Violations() Declared[Violation] { return decided.violations }
func (decided Decision) Reviewers() Declared[string]     { return decided.reviewers }
func (decided Decision) FailClosed() bool                { return decided.failClosed }
func (decided Decision) FailurePosture() FailurePosture  { return decided.posture }

type statedDecision struct {
	InputWellformed       bool                `json:"input_wellformed"`
	SchemaVersionAccepted int                 `json:"schema_version_accepted"`
	Effect                string              `json:"effect"`
	Violations            Declared[Violation] `json:"violations"`
	Reviewers             Declared[string]    `json:"reviewers"`
	FailClosed            *bool               `json:"fail_closed"`
}

func UndecidedDecision() Decision { return Decision{outcome: OutcomeUndefined} }

func ParseDecision(stated []byte) (Decision, error) {
	var written statedDecision
	if err := json.Unmarshal(stated, &written); err != nil {
		return Decision{}, fmt.Errorf("%w: %w", ErrMalformedDecision, err)
	}
	effect, err := ParseEffect(written.Effect)
	if err != nil {
		return Decision{}, fmt.Errorf("%w: %w", ErrMalformedDecision, err)
	}
	for _, violation := range written.Violations {
		if _, err := ParseEnforcementSeverity(string(violation.Severity)); err != nil {
			return Decision{}, fmt.Errorf("%w: %w", ErrMalformedDecision, err)
		}
	}
	if _, err := ParseSchemaMajor(written.SchemaVersionAccepted); err != nil {
		return Decision{}, err
	}
	if !written.InputWellformed {
		undecided := UndecidedDecision()
		if written.FailClosed != nil {
			undecided.posture = FailurePostureOpen
			if *written.FailClosed {
				undecided.posture = FailurePostureClosed
			}
		}
		return undecided, nil
	}
	return Decision{
		outcome:    OutcomeDecided,
		effect:     effect,
		violations: written.Violations,
		reviewers:  written.Reviewers,
		failClosed: written.FailClosed != nil && *written.FailClosed,
	}, nil
}
