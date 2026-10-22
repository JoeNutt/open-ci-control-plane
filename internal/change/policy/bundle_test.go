// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

const engineStartBound = 10 * time.Second

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

func TestThePreviousRevisionKeepsDecidingAfterARefusedOne(t *testing.T) {
	active := activePolicyLoggingTo(t, slog.DiscardHandler)
	recorded := policyRepositoryAt(t, firstPolicyCommit, liveFiles(t))
	recordCommit(t, recorded, secondPolicyCommit, liveFilesWithBinding(t, liveBindingWithAlias(t)))
	reader := readerOver(t, recorded, active)
	if err := reader.ReadRevision(context.Background()); err != nil {
		t.Fatalf("read the first head: %v", err)
	}
	engine := structuralEngineServedBy(t, policy.BundleHandler{Active: active})
	recorded.heads.Heads[readRepository(t)] = policyCommit(t, secondPolicyCommit)
	if err := reader.ReadRevision(context.Background()); !errors.Is(err, policy.ErrAliasOrAnchor) {
		t.Fatalf("an aliased binding gave %v, want ErrAliasOrAnchor", err)
	}

	decided, err := engine.Decide(context.Background(), domain.Document{
		PR:   domain.PullRequest{Repository: liveRepository},
		Diff: domain.Diff{ModuleGraphDelta: domain.ModuleGraphDelta{EdgesAdded: domain.Declared[domain.ModuleEdge]{}}},
	})

	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if decided.Outcome() != domain.OutcomeDecided || decided.Revision().String() != firstPolicyCommit {
		t.Errorf("outcome %q at %q, want a decision at the previous revision %s", decided.Outcome(), decided.Revision(), firstPolicyCommit)
	}
}

func TestTheStructuralDecisionIsUndefinedWhenNoValidPolicyFileWasEverRead(t *testing.T) {
	active := activePolicyLoggingTo(t, slog.DiscardHandler)
	aliased := policyRepositoryAt(t, firstPolicyCommit, liveFilesWithBinding(t, liveBindingWithAlias(t)))
	if err := readerOver(t, aliased, active).ReadRevision(context.Background()); !errors.Is(err, policy.ErrAliasOrAnchor) {
		t.Fatalf("a cold start with an aliased binding gave %v, want ErrAliasOrAnchor", err)
	}
	engine := structuralEngineServedBy(t, policy.BundleHandler{Active: active})

	decided, err := engine.Decide(context.Background(), domain.Document{
		PR:   domain.PullRequest{Repository: liveRepository},
		Diff: domain.Diff{ModuleGraphDelta: domain.ModuleGraphDelta{EdgesAdded: domain.Declared[domain.ModuleEdge]{}}},
	})

	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if decided.Outcome() != domain.OutcomeUndefined {
		t.Errorf("a cold start decided %q effect %q, want undefined", decided.Outcome(), decided.Effect())
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
	starting, cancel := context.WithTimeout(context.Background(), engineStartBound)
	t.Cleanup(cancel)
	engine, err := opa.ParseEngine(starting, opa.Settings{Family: domain.PolicyFamilyStructural, BundleService: service})
	if err != nil {
		t.Fatalf("parse engine: %v", err)
	}
	t.Cleanup(func() { engine.Stop(context.Background()) })
	return engine
}
