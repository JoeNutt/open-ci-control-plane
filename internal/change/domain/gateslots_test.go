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
	mutex   sync.Mutex
	running int
	most    int
	entered int
}

func (engine *occupyingEngine) Decide(ctx context.Context, _ domain.Document) (domain.Decision, error) {
	engine.mutex.Lock()
	engine.running++
	engine.entered++
	engine.most = max(engine.most, engine.running)
	engine.mutex.Unlock()

	<-ctx.Done()

	engine.mutex.Lock()
	engine.running--
	engine.mutex.Unlock()
	return domain.Decision{}, ctx.Err()
}

func TestNoMoreEvaluationsRunAtOnceThanTheSlotsAndTheNextOneWaits(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	engine := &occupyingEngine{}
	gate, err := domain.ParseGate(map[domain.PolicyFamily]domain.FamilyEngine{
		domain.PolicyFamilyStructural: engine,
	})
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}
	slots := domain.EvaluationSlots()

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
