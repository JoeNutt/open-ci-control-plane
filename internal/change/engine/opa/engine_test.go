// SPDX-License-Identifier: GPL-3.0-or-later

package opa_test

import (
	"context"
	"testing"

	sdktest "github.com/open-policy-agent/opa/v1/sdk/test"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa"
)

const decidingPolicy = `package ci.security

decision := {
	"input_wellformed": true,
	"schema_version_accepted": 1,
	"effect": "block",
	"violations": [{
		"rule_id": "AUTHZ-1",
		"message": "auth middleware changed with no authz test",
		"severity": "high",
		"owning_team": "security-platform",
		"docs_url": "https://example.test/policy/AUTHZ-1"
	}],
	"reviewers": ["acme/security-platform"],
	"fail_closed": true
}
`

func engineFor(t *testing.T, policy string, family domain.PolicyFamily) *opa.Engine {
	t.Helper()

	server := sdktest.MustNewServer(sdktest.MockBundle(
		"/bundles/"+string(family), map[string]string{"policy.rego": policy}))
	t.Cleanup(server.Stop)

	engine, err := opa.ParseEngine(context.Background(), opa.Settings{
		Family:        family,
		BundleService: server.URL(),
	})
	if err != nil {
		t.Fatalf("parse engine: %v", err)
	}
	t.Cleanup(func() { engine.Stop(context.Background()) })
	return engine
}

func TestAnEngineReturnsTheDecisionItsFamilyPublishes(t *testing.T) {
	engine := engineFor(t, decidingPolicy, domain.PolicyFamilySecurity)

	decided, err := engine.Decide(context.Background(), domain.Document{})

	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if decided.Outcome() != domain.OutcomeDecided {
		t.Fatalf("outcome is %q, want %q", decided.Outcome(), domain.OutcomeDecided)
	}
	if decided.Effect() != domain.EffectBlock {
		t.Errorf("effect is %q, want %q", decided.Effect(), domain.EffectBlock)
	}
	if len(decided.Violations()) != 1 {
		t.Errorf("violations are %v, want one", decided.Violations())
	}
}
