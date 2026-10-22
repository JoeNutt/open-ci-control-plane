// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa"
)

type stoppableEngine interface {
	domain.FamilyEngine
	Stop(ctx context.Context)
}

type engineStarter func(ctx context.Context, family domain.PolicyFamily) (stoppableEngine, error)

func bundledEngineStarter(bundleService string) engineStarter {
	return func(ctx context.Context, family domain.PolicyFamily) (stoppableEngine, error) {
		engine, err := opa.ParseEngine(ctx, opa.Settings{Family: family, BundleService: bundleService})
		if err != nil {
			return nil, err
		}
		return engine, nil
	}
}

type startedEngines map[domain.PolicyFamily]stoppableEngine

func startedEnginesFor(
	ctx context.Context, families []domain.PolicyFamily, start engineStarter,
) (startedEngines, error) {
	engines := make(startedEngines, len(families))
	for _, family := range families {
		engine, err := start(ctx, family)
		if err != nil {
			engines.stop(context.WithoutCancel(ctx))
			return nil, err
		}
		engines[family] = engine
	}
	return engines, nil
}

func (engines startedEngines) gate() (domain.Gate, error) {
	deciding := make(map[domain.PolicyFamily]domain.FamilyEngine, len(engines))
	for family, engine := range engines {
		deciding[family] = engine
	}
	return domain.ParseGate(deciding)
}

func (engines startedEngines) stop(ctx context.Context) {
	for _, engine := range engines {
		engine.Stop(ctx)
	}
}
