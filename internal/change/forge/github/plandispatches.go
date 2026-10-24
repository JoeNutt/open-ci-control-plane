// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"unicode/utf8"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var _ domain.PlanDispatcher = PlanDispatches{}

const planSchemaVersion = 1

type PlanDispatches struct {
	forge connection
	pause pauseFunc
}

func ParsePlanDispatches(settings Settings) (PlanDispatches, error) {
	forge, err := parseCredentialsConnection(settings)
	if err != nil {
		return PlanDispatches{}, err
	}
	return PlanDispatches{forge: forge, pause: waitFor}, nil
}

type planPayload struct {
	SchemaVersion   int              `json:"schema_version"`
	DecisionID      string           `json:"decisionId"`
	Repository      string           `json:"repository"`
	HeadSHA         string           `json:"headSha"`
	Tests           []testsPayload   `json:"tests"`
	AlwaysRunSuites []string         `json:"alwaysRunSuites"`
	FullMatrix      bool             `json:"fullMatrix"`
	TriggerReason   string           `json:"triggerReason"`
	Shadow          bool             `json:"shadow"`
	BundleRevision  string           `json:"bundleRevision"`
	Mode            planModePayload  `json:"mode"`
	Recorded        *recordedPayload `json:"recorded,omitempty"`
}

type testsPayload struct {
	Package string   `json:"package"`
	Names   []string `json:"names"`
}

type planModePayload struct {
	Selection    string `json:"selection"`
	Minimisation string `json:"minimisation"`
}

type recordedPayload struct {
	Selected  []string `json:"selected"`
	Minimised []string `json:"minimised"`
}

type dispatchInputs struct {
	Plan string `json:"plan"`
}

type workflowDispatch struct {
	Ref    string         `json:"ref"`
	Inputs dispatchInputs `json:"inputs"`
}

func (dispatches PlanDispatches) DispatchPlan(ctx context.Context, plan domain.TestPlan) error {
	dispatching := func(bounded context.Context) error { return dispatches.dispatch(bounded, plan) }
	return withinAttempts(ctx, dispatches.pause, dispatching)
}

func (dispatches PlanDispatches) dispatch(ctx context.Context, plan domain.TestPlan) error {
	branch, err := defaultBranchOf(ctx, dispatches.forge, plan.Repository)
	if err != nil {
		return fmt.Errorf("read the default branch of %s: %w", plan.Repository, err)
	}
	encoded, err := boundedEncodingOf(plan)
	if err != nil {
		return fmt.Errorf("encode the plan for %s: %w", plan.Repository, err)
	}
	if err := delivered(ctx, dispatches.forge, dispatchOf(plan, branch, string(encoded))); err != nil {
		return fmt.Errorf("dispatch the plan for %s at %s: %w", plan.Repository, plan.Head, err)
	}
	return nil
}

func boundedEncodingOf(plan domain.TestPlan) ([]byte, error) {
	encoded, err := json.Marshal(payloadOf(plan))
	if err != nil || utf8.RuneCount(encoded) <= domain.MaxPlanCharacters {
		return encoded, err
	}
	return json.Marshal(payloadOf(plan.AsTooLarge()))
}

func payloadOf(plan domain.TestPlan) planPayload {
	return planPayload{
		SchemaVersion:   planSchemaVersion,
		DecisionID:      plan.DecisionID.String(),
		Repository:      plan.Repository.String(),
		HeadSHA:         plan.Head.String(),
		Tests:           testsPayloadOf(plan.Tests),
		AlwaysRunSuites: namesOf(plan.AlwaysRun),
		FullMatrix:      plan.FullMatrix,
		TriggerReason:   string(plan.Trigger),
		BundleRevision:  plan.BundleRevision.String(),
		Shadow:          plan.Modes.Selection == domain.TestModeShadow,
		Mode:            planModePayload{Selection: plan.Modes.Selection.String(), Minimisation: plan.Modes.Minimisation.String()},
		Recorded:        recordedOf(plan.Recorded),
	}
}

func recordedOf(recorded domain.RecordedSets) *recordedPayload {
	if !recorded.IsRecorded {
		return nil
	}
	return &recordedPayload{Selected: packageNamesOf(recorded.Selected), Minimised: packageNamesOf(recorded.Minimised)}
}

func testsPayloadOf(tests []domain.PackageTests) []testsPayload {
	entries := make([]testsPayload, 0, len(tests))
	for _, entry := range tests {
		entries = append(entries, testsPayload{Package: entry.Package().String(), Names: namesOf(entry.Names())})
	}
	return entries
}

func packageNamesOf(tests []domain.PackageTests) []string {
	names := make([]string, 0, len(tests))
	for _, entry := range tests {
		names = append(names, entry.Package().String())
	}
	return names
}

func namesOf[Named fmt.Stringer](suites []Named) []string {
	names := make([]string, 0, len(suites))
	for _, suite := range suites {
		names = append(names, suite.String())
	}
	return names
}

func dispatchOf(plan domain.TestPlan, branch, encodedPlan string) exchange {
	return exchange{
		method:    http.MethodPost,
		endpoint:  workflowDispatchEndpoint(plan),
		sent:      workflowDispatch{Ref: branch, Inputs: dispatchInputs{Plan: encodedPlan}},
		succeeded: http.StatusNoContent,
	}
}

func workflowDispatchEndpoint(plan domain.TestPlan) string {
	return repositoryEndpoint(plan.Repository) + "/actions/workflows/" + url.PathEscape(plan.Workflow.String()) + "/dispatches"
}
