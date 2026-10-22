// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

type FamilyEngine interface {
	Decide(ctx context.Context, document Document) (Decision, error)
}

const EvaluationDeadline = 250 * time.Millisecond

var ErrEngineFault = errors.New("policy engine failed to evaluate")

var ErrEmptyGate = errors.New("gate is configured with no policy family")

var ErrIncompleteGate = errors.New("gate is configured with a family that has no engine")

type FamilyDecision struct {
	Family   PolicyFamily
	Decision Decision
}

func (decided FamilyDecision) FailurePosture() FailurePosture {
	if declared := decided.Decision.FailurePosture(); declared != "" {
		return declared
	}
	return decided.Family.FailurePosture()
}

type Gate struct {
	families []PolicyFamily
	engines  map[PolicyFamily]FamilyEngine
}

func ParseGate(engines map[PolicyFamily]FamilyEngine) (Gate, error) {
	if len(engines) == 0 {
		return Gate{}, ErrEmptyGate
	}
	held, err := heldEngines(engines)
	if err != nil {
		return Gate{}, err
	}
	return Gate{families: configuredIn(held), engines: held}, nil
}

func heldEngines(configured map[PolicyFamily]FamilyEngine) (map[PolicyFamily]FamilyEngine, error) {
	held := make(map[PolicyFamily]FamilyEngine, len(configured))
	for stated, engine := range configured {
		family, err := ParsePolicyFamily(string(stated))
		if err != nil {
			return nil, err
		}
		if engine == nil {
			return nil, fmt.Errorf("%w: %q", ErrIncompleteGate, family)
		}
		held[family] = engine
	}
	return held, nil
}

func (gate Gate) DecisionsFor(ctx context.Context, document Document) []FamilyDecision {
	decisions := make([]FamilyDecision, 0, len(gate.families))
	for _, family := range gate.families {
		decisions = append(decisions, FamilyDecision{
			Family:   family,
			Decision: gate.decisionFor(ctx, family, document),
		})
	}
	return decisions
}

func configuredIn(engines map[PolicyFamily]FamilyEngine) []PolicyFamily {
	return slices.DeleteFunc(DeclaredPolicyFamilies(), func(family PolicyFamily) bool {
		_, isConfigured := engines[family]
		return !isConfigured
	})
}

func (gate Gate) decisionFor(
	ctx context.Context, family PolicyFamily, document Document,
) Decision {
	evaluating, cancel := context.WithTimeout(ctx, EvaluationDeadline)
	defer cancel()
	decided, err := gate.engines[family].Decide(evaluating, document)
	if isPastDeadline(evaluating, err) {
		return Decision{outcome: OutcomeDeadlineExceeded}
	}
	if err != nil {
		return Decision{outcome: OutcomeEngineError}
	}
	return decided
}

func isPastDeadline(ctx context.Context, err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded)
}
