// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

type Outcome string

const (
	OutcomeDecided          Outcome = "decided"
	OutcomeUndefined        Outcome = "undefined"
	OutcomeEngineError      Outcome = "engine_error"
	OutcomeDeadlineExceeded Outcome = "deadline_exceeded"
)

var declaredOutcomes = []Outcome{
	OutcomeDecided,
	OutcomeUndefined,
	OutcomeEngineError,
	OutcomeDeadlineExceeded,
}

var ErrUnknownOutcome = errors.New("unknown evaluation outcome")

func ParseOutcome(name string) (Outcome, error) {
	return parseDeclaration(name, declaredOutcomes, ErrUnknownOutcome)
}

func (outcome Outcome) CarriesVerdict() bool { return outcome == OutcomeDecided }
