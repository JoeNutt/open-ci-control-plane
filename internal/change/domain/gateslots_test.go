// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"context"
	"runtime"
	"sync"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type occupyingEngine struct {
	arrivals chan struct{}
	mutex    sync.Mutex
	running  int
	most     int
	entered  int
}

func (engine *occupyingEngine) Decide(ctx context.Context, _ domain.Document) (domain.Decision, error) {
	engine.mutex.Lock()
	engine.running++
	engine.entered++
	engine.most = max(engine.most, engine.running)
	engine.mutex.Unlock()
	engine.arrivals <- struct{}{}

	<-ctx.Done()

	engine.mutex.Lock()
	engine.running--
	engine.mutex.Unlock()
	return domain.Decision{}, ctx.Err()
}

func TestNoMoreEvaluationsRunAtOnceThanTheSlotsAndTheNextOneWaits(t *testing.T) {
	gate, engine, slots := gateWithOneProcessor(t)

	var evaluations sync.WaitGroup
	for range slots + 1 {
		evaluations.Go(func() { gate.DecisionsFor(context.Background(), domain.Document{}) })
	}
	evaluations.Wait()

	if slots != 2 {
		t.Errorf("slots at GOMAXPROCS 1 are %d, want 2", slots)
	}
	if engine.most != slots {
		t.Errorf("at most %d evaluations ran at once, want %d", engine.most, slots)
	}
	if engine.entered != slots+1 {
		t.Errorf("%d of %d evaluations ran, want every one", engine.entered, slots+1)
	}
}

func TestAnEvaluationWhoseCallerStopsWaitingForASlotIsAPolicyFaultNotAnAdmit(t *testing.T) {
	gate, engine, slots := gateWithOneProcessor(t)
	var occupants sync.WaitGroup
	for range slots {
		occupants.Go(func() { gate.DecisionsFor(context.Background(), domain.Document{}) })
	}
	for range slots {
		<-engine.arrivals
	}
	abandoned, abandon := context.WithCancel(context.Background())
	abandon()

	decided := gate.DecisionsFor(abandoned, domain.Document{})

	occupants.Wait()
	if decided[0].Decision.Outcome() != domain.OutcomeDeadlineExceeded {
		t.Errorf("outcome is %q, want %q", decided[0].Decision.Outcome(), domain.OutcomeDeadlineExceeded)
	}
	if engine.entered != slots {
		t.Errorf("%d evaluations reached the engine, want only the %d occupants", engine.entered, slots)
	}
}

func gateWithOneProcessor(t *testing.T) (domain.Gate, *occupyingEngine, int) {
	t.Helper()

	previous := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	slots := domain.EvaluationSlots()
	engine := &occupyingEngine{arrivals: make(chan struct{}, slots+1)}
	gate, err := domain.ParseGate(map[domain.PolicyFamily]domain.FamilyEngine{
		domain.PolicyFamilyStructural: engine,
	})
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}
	return gate, engine, slots
}
