// SPDX-License-Identifier: GPL-3.0-or-later

// Package opa evaluates one policy family in its own embedded engine instance.
package opa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/v1/sdk"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrIncompleteSettings = errors.New("engine settings are missing a part the engine cannot work without")

type Settings struct {
	Family        domain.PolicyFamily
	BundleService string
}

type Engine struct {
	family domain.PolicyFamily
	opa    *sdk.OPA
}

func ParseEngine(ctx context.Context, settings Settings) (*Engine, error) {
	if _, err := domain.ParsePolicyFamily(string(settings.Family)); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrIncompleteSettings, err)
	}
	if settings.BundleService == "" {
		return nil, fmt.Errorf("%w: no bundle service to load %q from",
			ErrIncompleteSettings, settings.Family)
	}
	configured, err := json.Marshal(configurationFor(settings))
	if err != nil {
		return nil, fmt.Errorf("marshal %q configuration: %w", settings.Family, err)
	}
	started, err := sdk.New(ctx, sdk.Options{
		ID:     string(settings.Family),
		Config: bytes.NewReader(configured),
	})
	if err != nil {
		return nil, fmt.Errorf("start the %q engine: %w", settings.Family, err)
	}
	return &Engine{family: settings.Family, opa: started}, nil
}

func (engine *Engine) Stop(ctx context.Context) { engine.opa.Stop(ctx) }

func (engine *Engine) Decide(
	ctx context.Context, document domain.Document,
) (domain.Decision, error) {
	decided, err := engine.decisionFor(ctx, document)
	if err != nil {
		return domain.Decision{}, fmt.Errorf("the %q family: %w", engine.family, err)
	}
	return decided, nil
}

func (engine *Engine) decisionFor(
	ctx context.Context, document domain.Document,
) (domain.Decision, error) {
	answered, err := engine.opa.Decision(ctx, sdk.DecisionOptions{
		Path:                decisionPathFor(engine.family),
		Input:               document,
		StrictBuiltinErrors: true,
	})
	if sdk.IsUndefinedErr(err) {
		return domain.UndecidedDecision(), nil
	}
	if err != nil {
		return domain.Decision{}, fmt.Errorf("%w: %v", domain.ErrEngineFault, err)
	}
	stated, err := json.Marshal(answered.Result)
	if err != nil {
		return domain.Decision{}, fmt.Errorf("re-encode the decision: %w", err)
	}
	return domain.ParseDecision(stated)
}

func decisionPathFor(family domain.PolicyFamily) string {
	return strings.ReplaceAll(strings.TrimPrefix(family.DecisionQuery(), "data."), ".", "/")
}

type configuration struct {
	Services map[string]service `json:"services"`
	Bundles  map[string]bundle  `json:"bundles"`
}

type service struct {
	URL string `json:"url"`
}

type bundle struct {
	Service  string `json:"service"`
	Resource string `json:"resource"`
}

func configurationFor(settings Settings) configuration {
	family := string(settings.Family)
	return configuration{
		Services: map[string]service{family: {URL: settings.BundleService}},
		Bundles: map[string]bundle{
			family: {Service: family, Resource: "/bundles/" + family},
		},
	}
}
