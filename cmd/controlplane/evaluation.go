// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/httpapi"
)

var errEvaluationUnwired = errors.New(
	"evaluation is not wired: the forge credential arrives with #120 and the policy bundle with step 4")

type unwiredEvaluation struct{}

func (unwiredEvaluation) Execute(context.Context, domain.PullRequestLocator, domain.DecisionID) error {
	return errEvaluationUnwired
}

func unwiredGates(installations []int64) map[int64]httpapi.ChangeGate {
	gates := make(map[int64]httpapi.ChangeGate, len(installations))
	for _, installation := range installations {
		gates[installation] = unwiredEvaluation{}
	}
	return gates
}
