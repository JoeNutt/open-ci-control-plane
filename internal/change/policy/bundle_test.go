// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

func TestTheStructuralEngineDecidesWithTheRevisionTheReaderAccepted(t *testing.T) {
	active := activePolicyLoggingTo(t, slog.DiscardHandler)
	if err := readerOver(t, policyRepositoryAt(t, firstPolicyCommit, liveFiles(t)), active).ReadRevision(context.Background()); err != nil {
		t.Fatalf("read revision: %v", err)
	}
	engine := structuralEngineServedBy(t, policy.BundleHandler{Active: active})

	decided, err := engine.Decide(context.Background(), domain.Document{
		PR:   domain.PullRequest{Repository: liveRepository},
		Diff: domain.Diff{ModuleGraphDelta: domain.ModuleGraphDelta{EdgesAdded: domain.Declared[domain.ModuleEdge]{}}},
	})

	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if decided.Outcome() != domain.OutcomeDecided || decided.Effect() != domain.EffectAllow {
		t.Errorf("outcome %q effect %q, want a decided allow", decided.Outcome(), decided.Effect())
	}
	if decided.Revision().String() != firstPolicyCommit {
		t.Errorf("the engine decided with revision %q, want the policy commit %s", decided.Revision(), firstPolicyCommit)
	}
}

func structuralEngineServedBy(t *testing.T, bundles policy.BundleHandler) *opa.Engine {
	t.Helper()

	server := httptest.NewServer(bundles)
	t.Cleanup(server.Close)
	service, err := opa.ParseBundleService(server.URL)
	if err != nil {
		t.Fatalf("parse bundle service: %v", err)
	}
	engine, err := opa.ParseEngine(context.Background(), opa.Settings{Family: domain.PolicyFamilyStructural, BundleService: service})
	if err != nil {
		t.Fatalf("parse engine: %v", err)
	}
	t.Cleanup(func() { engine.Stop(context.Background()) })
	return engine
}
