// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

const (
	twoBindingsCommit = "b0b0000000000000000000000000000000000b0b"
	engineStartBound  = 10 * time.Second
)

var twoBindingsPolicyFiles = []domain.PolicyFile{
	{Path: "catalog/clean-architecture.yaml", Content: []byte(`schema_version: 1
name: clean-architecture
rule: direction
direction: lower_order_may_not_import_higher
groups:
  - { name: domain, order: 0 }
  - { name: adapter, order: 1 }
`)},
	{Path: "catalog/mvvm.yaml", Content: []byte(`schema_version: 1
name: mvvm
rule: direction
direction: lower_order_may_not_import_higher
groups:
  - { name: model, order: 0 }
  - { name: view, order: 1 }
`)},
	{Path: "bindings/acme/payments.yaml", Content: []byte(`schema_version: 1
repository: acme/payments
policies:
  - use: clean-architecture
    import_roots:
      - { prefix: example.com/app, path: "" }
    paths:
      domain: ["internal/domain/**"]
      adapter: ["internal/forge/**"]
tests:
  workflow: open-ci-tests.yml
`)},
	{Path: "bindings/acme/mobile-app.yaml", Content: []byte(`schema_version: 1
repository: acme/mobile-app
policies:
  - use: mvvm
    import_roots:
      - { prefix: example.com/app, path: "" }
    paths:
      model: ["internal/domain/**"]
      view: ["internal/view/**"]
tests:
  workflow: open-ci-tests.yml
`)},
}

func TestEachRepositoryIsJudgedOnlyByThePoliciesItsOwnBindingNames(t *testing.T) {
	engine := structuralEngineServedFrom(t, twoBindingsPolicyFiles)
	importsForge := importEdit{
		path: "internal/domain/order.go",
		base: "package domain\n", head: "package domain\n\nimport \"example.com/app/internal/forge\"\n",
	}
	importsView := importEdit{
		path: "internal/domain/order.go",
		base: "package domain\n", head: "package domain\n\nimport \"example.com/app/internal/view\"\n",
	}
	cases := map[string]struct {
		repository string
		edit       importEdit
		want       domain.Conclusion
	}{
		"clean-architecture refuses in payments":  {"acme/payments", importsForge, domain.ConclusionFailure},
		"mvvm does not apply to payments":         {"acme/payments", importsView, domain.ConclusionSuccess},
		"mvvm refuses in the mobile app":          {"acme/mobile-app", importsView, domain.ConclusionFailure},
		"clean-architecture does not apply to it": {"acme/mobile-app", importsForge, domain.ConclusionSuccess},
	}

	for form, judged := range cases {
		t.Run(form, func(t *testing.T) {
			change := changeAddingImport(t, locatorOf(t, judged.repository, 412), judged.edit)

			run := structuralCheckRunAfterGating(t, engine, change)

			if run.Conclusion != judged.want {
				t.Errorf("conclusion is %q with %+v, want %q", run.Conclusion, run.Violations, judged.want)
			}
		})
	}
}

func structuralEngineServedFrom(t *testing.T, files []domain.PolicyFile) *opa.Engine {
	t.Helper()

	active, err := policy.ParseActivePolicy(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("parse active policy: %v", err)
	}
	if err := active.Offer(commitSHAOf(t, twoBindingsCommit), files); err != nil {
		t.Fatalf("offer the policy revision: %v", err)
	}
	server := httptest.NewServer(policy.BundleHandler{Active: active})
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
