// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"strings"
	"testing"

	sdktest "github.com/open-policy-agent/opa/v1/sdk/test"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa/structural"
)

const otherRepositoryBinding = `"acme/mobile-app": {"policies": [{
	"use": "mvvm",
	"import_roots": [{"prefix": "example.com/mobile", "path": ""}],
	"paths": {"model": ["app/model/**"]}
}]}`

const mvvmCatalog = `"mvvm": {"rule": "direction", "direction": "lower_order_may_not_import_higher",
	"groups": [{"name": "model", "order": 0}]}`

func TestEachRepositoryTheStructuralPolicyCannotJudgeFailsNamingWhy(t *testing.T) {
	cases := map[string]struct {
		policyData string
		names      string
	}{
		"an unbound repository": {
			policyData: `{"catalog": {` + mvvmCatalog + `}, "bindings": {` + otherRepositoryBinding + `}}`,
			names:      "acme/payments has no binding",
		},
		"a binding naming a policy absent from the catalog": {
			policyData: `{"catalog": {}, "bindings": {"acme/payments": {"policies": [{
				"use": "mvvm", "import_roots": [], "paths": {}
			}]}}}`,
			names: "names policy mvvm, absent from the catalog",
		},
	}

	for form, policy := range cases {
		t.Run(form, func(t *testing.T) {
			run := structuralCheckRunAfter(t, structuralEngineServing(t, policy.policyData))

			if run.Conclusion != domain.ConclusionFailure {
				t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
			}
			if len(run.Violations) != 1 || !strings.Contains(run.Violations[0].Message, policy.names) {
				t.Errorf("the check run does not name %q: %+v", policy.names, run.Violations)
			}
		})
	}
}

func structuralEngineServing(t *testing.T, policyData string) *opa.Engine {
	t.Helper()

	family := domain.PolicyFamilyStructural
	bundleFiles := structural.Modules()
	bundleFiles["data.json"] = `{"ci": {"policy": ` + policyData + `}}`
	bundleFiles[".manifest"] = `{"revision": "` + revisionForTest(t).String() + `"}`
	server := sdktest.MustNewServer(sdktest.MockBundle("/bundles/"+string(family), bundleFiles))
	t.Cleanup(server.Stop)

	service, err := opa.ParseBundleService(server.URL())
	if err != nil {
		t.Fatalf("parse bundle service: %v", err)
	}
	engine, err := opa.ParseEngine(context.Background(), opa.Settings{Family: family, BundleService: service})
	if err != nil {
		t.Fatalf("parse engine: %v", err)
	}
	t.Cleanup(func() { engine.Stop(context.Background()) })
	return engine
}
