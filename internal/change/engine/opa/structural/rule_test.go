// SPDX-License-Identifier: GPL-3.0-or-later

package structural_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	sdktest "github.com/open-policy-agent/opa/v1/sdk/test"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa/structural"
)

const policyRevision = "c0ffee0000000000000000000000000000c0ffee"

func TestTheEmbeddedRuleRefusesAForbiddenAddedEdgeThroughTheEngine(t *testing.T) {
	policyData, err := os.ReadFile(filepath.Join("testdata", "ci", "policy", "data.json"))
	if err != nil {
		t.Fatalf("read the policy fixture: %v", err)
	}
	bundleFiles := structural.Modules()
	bundleFiles["data.json"] = `{"ci": {"policy": ` + string(policyData) + `}}`
	bundleFiles[".manifest"] = `{"revision": "` + policyRevision + `"}`
	server := sdktest.MustNewServer(sdktest.MockBundle("/bundles/structural", bundleFiles))
	t.Cleanup(server.Stop)
	engine, err := opa.ParseEngine(context.Background(), opa.Settings{
		Family:        domain.PolicyFamilyStructural,
		BundleService: server.URL(),
	})
	if err != nil {
		t.Fatalf("parse engine: %v", err)
	}
	t.Cleanup(func() { engine.Stop(context.Background()) })

	decided, err := engine.Decide(context.Background(), domain.Document{
		PR: domain.PullRequest{Repository: "JoeNutt/open-ci-control-plane"},
		Diff: domain.Diff{ModuleGraphDelta: domain.ModuleGraphDelta{
			EdgesAdded: domain.Declared[domain.ModuleEdge]{{
				From: "internal/change/domain/gate.go",
				To:   "github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github",
			}},
		}},
	})

	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if decided.Outcome() != domain.OutcomeDecided || decided.Effect() != domain.EffectBlock {
		t.Fatalf("outcome %q effect %q, want %q %q",
			decided.Outcome(), decided.Effect(), domain.OutcomeDecided, domain.EffectBlock)
	}
	if len(decided.Violations()) != 1 || decided.Violations()[0].RuleID != "clean-architecture" {
		t.Errorf("violations are %+v, want one from clean-architecture", decided.Violations())
	}
	if decided.Revision().String() != policyRevision {
		t.Errorf("revision is %q, want %q", decided.Revision(), policyRevision)
	}
}
