// SPDX-License-Identifier: GPL-3.0-or-later

package structural_test

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	sdktest "github.com/open-policy-agent/opa/v1/sdk/test"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa/structural"
)

const policyRevision = "c0ffee0000000000000000000000000000c0ffee"

type policyFixture struct {
	Bindings map[string]struct {
		Policies []struct {
			Use string `json:"use"`
		} `json:"policies"`
	} `json:"bindings"`
}

type edgeFixture struct {
	DomainImportsForge domain.ModuleEdge `json:"domain_imports_forge"`
}

func TestTheEmbeddedRuleRefusesAForbiddenAddedEdgeThroughTheEngine(t *testing.T) {
	policyData := fixtureBytes(t, "ci", "policy")
	policy := decoded[policyFixture](t, policyData)
	repository := slices.Collect(maps.Keys(policy.Bindings))[0]
	engine := engineServing(t, policyData)

	decided, err := engine.Decide(context.Background(), domain.Document{
		PR: domain.PullRequest{Repository: repository},
		Diff: domain.Diff{ModuleGraphDelta: domain.ModuleGraphDelta{
			EdgesAdded: domain.Declared[domain.ModuleEdge]{
				decoded[edgeFixture](t, fixtureBytes(t, "edges")).DomainImportsForge,
			},
		}},
	})

	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if decided.Outcome() != domain.OutcomeDecided || decided.Effect() != domain.EffectBlock {
		t.Fatalf("outcome %q effect %q, want %q %q",
			decided.Outcome(), decided.Effect(), domain.OutcomeDecided, domain.EffectBlock)
	}
	boundPolicy := policy.Bindings[repository].Policies[0].Use
	if len(decided.Violations()) != 1 || decided.Violations()[0].RuleID != boundPolicy {
		t.Errorf("violations are %+v, want one from %q", decided.Violations(), boundPolicy)
	}
	if decided.Revision().String() != policyRevision {
		t.Errorf("revision is %q, want %q", decided.Revision(), policyRevision)
	}
}

func engineServing(t *testing.T, policyData []byte) *opa.Engine {
	t.Helper()

	bundleFiles := structural.Modules()
	bundleFiles["data.json"] = `{"ci": {"policy": ` + string(policyData) + `}}`
	bundleFiles[".manifest"] = `{"revision": "` + policyRevision + `"}`
	family := domain.PolicyFamilyStructural
	server := sdktest.MustNewServer(sdktest.MockBundle("/bundles/"+string(family), bundleFiles))
	t.Cleanup(server.Stop)

	engine, err := opa.ParseEngine(context.Background(), opa.Settings{Family: family, BundleService: server.URL()})
	if err != nil {
		t.Fatalf("parse engine: %v", err)
	}
	t.Cleanup(func() { engine.Stop(context.Background()) })
	return engine
}

func fixtureBytes(t *testing.T, dataPath ...string) []byte {
	t.Helper()

	read, err := os.ReadFile(filepath.Join(append(append([]string{"testdata"}, dataPath...), "data.json")...))
	if err != nil {
		t.Fatalf("read the %v fixture: %v", dataPath, err)
	}
	return read
}

func decoded[Shape any](t *testing.T, encoded []byte) Shape {
	t.Helper()

	var shaped Shape
	if err := json.Unmarshal(encoded, &shaped); err != nil {
		t.Fatalf("decode a fixture: %v", err)
	}
	return shaped
}
