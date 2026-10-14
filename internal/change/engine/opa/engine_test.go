// SPDX-License-Identifier: GPL-3.0-or-later

package opa_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	sdktest "github.com/open-policy-agent/opa/v1/sdk/test"
	"github.com/open-policy-agent/opa/v1/topdown"

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

const silentPolicy = `package ci.security

# No decision rule at all, so the query is undefined rather than false.
other := true
`

func TestAPolicyThatDoesNotDecideIsUndefinedNotAFault(t *testing.T) {
	engine := engineFor(t, silentPolicy, domain.PolicyFamilySecurity)

	decided, err := engine.Decide(context.Background(), domain.Document{})

	if err != nil {
		t.Fatalf("an undefined decision was reported as a fault: %v", err)
	}
	if decided.Outcome() != domain.OutcomeUndefined {
		t.Fatalf("outcome is %q, want %q", decided.Outcome(), domain.OutcomeUndefined)
	}
	if decided.Outcome().CarriesVerdict() {
		t.Fatal("an undefined outcome was allowed to carry a verdict")
	}
}

const nextMajorPolicy = `package ci.quality

decision := {
	"input_wellformed": true,
	"schema_version_accepted": 2,
	"effect": "allow",
	"violations": [],
	"reviewers": [],
	"fail_closed": false
}
`

func TestABundleAcceptingAnotherMajorIsAPolicyErrorOfItsFamily(t *testing.T) {
	engine := engineFor(t, nextMajorPolicy, domain.PolicyFamilyQuality)

	_, err := engine.Decide(context.Background(), domain.Document{})

	if !errors.Is(err, domain.ErrSchemaMajorMismatch) {
		t.Fatalf("want ErrSchemaMajorMismatch, got %v", err)
	}
	if !strings.Contains(err.Error(), string(domain.PolicyFamilyQuality)) {
		t.Errorf("the policy error does not name the %q family: %v",
			domain.PolicyFamilyQuality, err)
	}
}

const conflictingPolicy = `package ci.quality

decision := {"effect": "allow"}

decision := {"effect": "block"} if { input }
`

func TestAnEvaluationFaultCrossesThePortInTheDomainsTerms(t *testing.T) {
	engine := engineFor(t, conflictingPolicy, domain.PolicyFamilyQuality)

	_, err := engine.Decide(context.Background(), domain.Document{})

	if !errors.Is(err, domain.ErrEngineFault) {
		t.Fatalf("want ErrEngineFault, got %v", err)
	}
	var vendor *topdown.Error
	if errors.As(err, &vendor) {
		t.Errorf("an OPA error type crossed the port: %v", vendor)
	}
}
