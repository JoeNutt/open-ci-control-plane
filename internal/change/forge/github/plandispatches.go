// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var _ domain.PlanDispatcher = PlanDispatches{}

const planSchemaVersion = 1

type PlanDispatches struct {
	forge connection
}

func ParsePlanDispatches(settings Settings) (PlanDispatches, error) {
	forge, err := parseCredentialsConnection(settings)
	if err != nil {
		return PlanDispatches{}, err
	}
	return PlanDispatches{forge: forge}, nil
}

type planPayload struct {
	SchemaVersion   int      `json:"schema_version"`
	DecisionID      string   `json:"decisionId"`
	Repository      string   `json:"repository"`
	HeadSHA         string   `json:"headSha"`
	Suites          []string `json:"suites"`
	AlwaysRunSuites []string `json:"alwaysRunSuites"`
	FullMatrix      bool     `json:"fullMatrix"`
	TriggerReason   string   `json:"triggerReason"`
	Shadow          bool     `json:"shadow"`
	BundleRevision  string   `json:"bundleRevision"`
}

type dispatchInputs struct {
	Plan string `json:"plan"`
}

type workflowDispatch struct {
	Ref    string         `json:"ref"`
	Inputs dispatchInputs `json:"inputs"`
}

func (dispatches PlanDispatches) DispatchPlan(ctx context.Context, plan domain.TestPlan) error {
	branch, err := defaultBranchOf(ctx, dispatches.forge, plan.Repository)
	if err != nil {
		return fmt.Errorf("read the default branch of %s: %w", plan.Repository, err)
	}
	payload := planPayload{
		SchemaVersion:   planSchemaVersion,
		DecisionID:      plan.DecisionID.String(),
		Repository:      plan.Repository.String(),
		HeadSHA:         plan.Head.String(),
		Suites:          []string{},
		AlwaysRunSuites: []string{},
		FullMatrix:      plan.FullMatrix,
		TriggerReason:   string(plan.Trigger),
		BundleRevision:  plan.BundleRevision.String(),
	}
	for _, suite := range plan.Suites {
		payload.Suites = append(payload.Suites, suite.String())
	}
	for _, suite := range plan.AlwaysRun {
		payload.AlwaysRunSuites = append(payload.AlwaysRunSuites, suite.String())
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return delivered(ctx, dispatches.forge, exchange{
		method: http.MethodPost,
		endpoint: repositoryEndpoint(plan.Repository) + "/actions/workflows/" +
			url.PathEscape(plan.Workflow.String()) + "/dispatches",
		sent:      workflowDispatch{Ref: branch, Inputs: dispatchInputs{Plan: string(encoded)}},
		succeeded: http.StatusNoContent,
	})
}
