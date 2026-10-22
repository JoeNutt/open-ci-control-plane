// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa"
)

type startedEngines map[domain.PolicyFamily]*opa.Engine

func startedEnginesFor(
	ctx context.Context, families []domain.PolicyFamily, bundleService string,
) (startedEngines, error) {
	engines := startedEngines{}
	for _, family := range families {
		engine, err := opa.ParseEngine(ctx, opa.Settings{Family: family, BundleService: bundleService})
		if err != nil {
			return nil, err
		}
		engines[family] = engine
	}
	return engines, nil
}

func (engines startedEngines) gate() (domain.Gate, error) {
	deciding := map[domain.PolicyFamily]domain.FamilyEngine{}
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
