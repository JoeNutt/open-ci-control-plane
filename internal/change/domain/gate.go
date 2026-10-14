// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"context"
	"errors"
	"fmt"
)

type FamilyEngine interface {
	Decide(ctx context.Context, document Document) (Decision, error)
}

var ErrEngineFault = errors.New("policy engine failed to evaluate")

var ErrIncompleteGate = errors.New("gate is missing an engine for a declared family")

type FamilyDecision struct {
	Family   PolicyFamily
	Decision Decision
}

type Gate struct {
	engines map[PolicyFamily]FamilyEngine
}

func ParseGate(engines map[PolicyFamily]FamilyEngine) (Gate, error) {
	held := make(map[PolicyFamily]FamilyEngine, len(declaredPolicyFamilies))
	for _, family := range declaredPolicyFamilies {
		engine, present := engines[family]
		if !present || engine == nil {
			return Gate{}, fmt.Errorf("%w: %q", ErrIncompleteGate, family)
		}
		held[family] = engine
	}
	return Gate{engines: held}, nil
}

func (gate Gate) DecisionsFor(ctx context.Context, document Document) []FamilyDecision {
	decisions := make([]FamilyDecision, 0, len(declaredPolicyFamilies))
	for _, family := range declaredPolicyFamilies {
		decisions = append(decisions, FamilyDecision{
			Family:   family,
			Decision: gate.decisionFor(ctx, family, document),
		})
	}
	return decisions
}

func (gate Gate) decisionFor(
	ctx context.Context, family PolicyFamily, document Document,
) Decision {
	decided, err := gate.engines[family].Decide(ctx, document)
	if isPastDeadline(ctx, err) {
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
