// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

type FailurePosture string

const (
	FailurePostureClosed FailurePosture = "closed"
	FailurePostureOpen   FailurePosture = "open"
)

type PolicyFamily string

const (
	PolicyFamilyStructural PolicyFamily = "structural"
	PolicyFamilySecurity   PolicyFamily = "security"
	PolicyFamilyQuality    PolicyFamily = "quality"
	PolicyFamilyRouting    PolicyFamily = "routing"
)

var declaredPolicyFamilies = []PolicyFamily{
	PolicyFamilyStructural,
	PolicyFamilySecurity,
	PolicyFamilyQuality,
	PolicyFamilyRouting,
}

var failurePostures = map[PolicyFamily]FailurePosture{
	PolicyFamilyStructural: FailurePostureClosed,
	PolicyFamilySecurity:   FailurePostureClosed,
	PolicyFamilyQuality:    FailurePostureOpen,
	PolicyFamilyRouting:    FailurePostureOpen,
}

var ErrUnknownPolicyFamily = errors.New("unknown policy family")

func ParsePolicyFamily(name string) (PolicyFamily, error) {
	return parseDeclaration(name, declaredPolicyFamilies, ErrUnknownPolicyFamily)
}

func DeclaredPolicyFamilies() []PolicyFamily {
	return append([]PolicyFamily(nil), declaredPolicyFamilies...)
}

func (family PolicyFamily) DecisionQuery() string {
	return fmt.Sprintf("data.ci.%s.decision", family)
}

func (family PolicyFamily) FailurePosture() FailurePosture {
	posture, declared := failurePostures[family]
	if !declared {
		return FailurePostureClosed
	}
	return posture
}
