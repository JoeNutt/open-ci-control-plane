// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	sdktest "github.com/open-policy-agent/opa/v1/sdk/test"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa"
)

const undecidingStructuralPolicy = "package ci.structural\n"

func structuralBundleService(t *testing.T) opa.BundleService {
	t.Helper()
	server := sdktest.MustNewServer(sdktest.MockBundle(opa.BundleResourceFor(domain.PolicyFamilyStructural), map[string]string{
		"policy.rego": undecidingStructuralPolicy,
		".manifest":   `{"revision": "b0b0000000000000000000000000000000000b0b"}`,
	}))
	t.Cleanup(server.Stop)
	service, err := opa.ParseBundleService(server.URL())
	if err != nil {
		t.Fatalf("the test bundle server's address: %v", err)
	}
	return service
}

func TestAFamilyWhoseBundleNeverArrivesRefusesStartUpByTheDeadlineNamingIt(t *testing.T) {
	deadline := 500 * time.Millisecond
	starting, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	began := time.Now()

	_, err := startedEnginesFor(starting,
		[]domain.PolicyFamily{domain.PolicyFamilyStructural, domain.PolicyFamilySecurity},
		bundledEngineStarter(structuralBundleService(t)))

	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), string(domain.PolicyFamilySecurity)) {
		t.Errorf("the refusal is not the start-up deadline naming the security family: %v", err)
	}
	if waited := time.Since(began); waited > 4*deadline {
		t.Errorf("start-up waited %s, past its %s deadline", waited, deadline)
	}
}

func TestEnginesStartOnlyForTheConfiguredFamilies(t *testing.T) {
	engines := structuralEngines(t)

	started := slices.Collect(maps.Keys(engines))
	if !slices.Equal(started, []domain.PolicyFamily{domain.PolicyFamilyStructural}) {
		t.Errorf("engines started for %v, want the structural family only", started)
	}
}

func TestTheGateOfTheStartedEnginesDecidesOnlyTheConfiguredFamilies(t *testing.T) {
	gate, err := structuralEngines(t).gate()
	if err != nil {
		t.Fatalf("build the gate: %v", err)
	}

	decided := gate.DecisionsFor(context.Background(), domain.Document{})
	if len(decided) != 1 || decided[0].Family != domain.PolicyFamilyStructural {
		t.Errorf("the gate decided %v, want the structural family only", decided)
	}
}

func structuralEngines(t *testing.T) startedEngines {
	t.Helper()
	starting, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	engines, err := startedEnginesFor(starting, []domain.PolicyFamily{domain.PolicyFamilyStructural},
		bundledEngineStarter(structuralBundleService(t)))
	if err != nil {
		t.Fatalf("start the engines: %v", err)
	}
	t.Cleanup(func() { engines.stop(context.Background()) })
	return engines
}

type countedEngine struct{ stops *int }

func (countedEngine) Decide(context.Context, domain.Document) (domain.Decision, error) {
	return domain.Decision{}, errors.New("a counted engine does not decide")
}

func (engine countedEngine) Stop(context.Context) { *engine.stops++ }

func TestAnEngineStartedBeforeALaterOneFailsIsStopped(t *testing.T) {
	stops := 0
	refused := errors.New("the security bundle never arrived")
	start := func(_ context.Context, family domain.PolicyFamily) (stoppableEngine, error) {
		if family == domain.PolicyFamilySecurity {
			return nil, refused
		}
		return countedEngine{stops: &stops}, nil
	}

	_, err := startedEnginesFor(context.Background(),
		[]domain.PolicyFamily{domain.PolicyFamilyStructural, domain.PolicyFamilySecurity}, start)

	if !errors.Is(err, refused) {
		t.Errorf("start-up error is %v, want %v", err, refused)
	}
	if stops != 1 {
		t.Errorf("the structural engine was stopped %d times, want once", stops)
	}
}
