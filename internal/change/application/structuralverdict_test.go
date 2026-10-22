// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/extract/golang"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

const paymentsCleanArchitecture = `{
	"catalog": {"clean-architecture": {
		"rule": "direction", "direction": "lower_order_may_not_import_higher",
		"groups": [{"name": "domain", "order": 0}, {"name": "adapter", "order": 1}]
	}},
	"bindings": {"acme/payments": {"policies": [{
		"use": "clean-architecture",
		"import_roots": [{"prefix": "example.com/payments", "path": ""}],
		"paths": {"domain": ["internal/domain/**"], "adapter": ["internal/forge/**"]}
	}]}}
}`

func TestAnAddedForbiddenImportFailsListingTheFileTheImportAndBothGroups(t *testing.T) {
	const (
		importer = "internal/domain/order.go"
		imported = "example.com/payments/internal/forge"
	)
	build := buildAddingImport(t, importer, "package domain\n", "package domain\n\nimport \""+imported+"\"\n")

	run := structuralCheckRunBuiltBy(t, build)

	if run.Conclusion != domain.ConclusionFailure {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
	}
	want := importer + " imports " + imported + `: group "domain" may not import group "adapter"`
	if len(run.Violations) != 1 || run.Violations[0].Message != want {
		t.Errorf("the check run lists %+v, want one violation reading %q", run.Violations, want)
	}
}

func TestAnAddedImportTheDirectionAllowsSucceeds(t *testing.T) {
	build := buildAddingImport(t, "internal/forge/client.go",
		"package forge\n", "package forge\n\nimport \"example.com/payments/internal/domain\"\n")

	run := structuralCheckRunBuiltBy(t, build)

	if run.Conclusion != domain.ConclusionSuccess || len(run.Violations) != 0 {
		t.Errorf("conclusion %q with %+v, want %q and none", run.Conclusion, run.Violations, domain.ConclusionSuccess)
	}
}

func structuralCheckRunBuiltBy(t *testing.T, build application.BuildDocument) domain.CheckRun {
	t.Helper()

	published := &memory.CheckRuns{}
	gateChangeBuiltBy(t, build, structuralEngineServing(t, paymentsCleanArchitecture), published)
	run, present := published.PublishedOn(
		gatedLocator(t), commitSHAOf(t, recordedHeadSHA))[structuralCheckRunName]
	if !present {
		t.Fatalf("no structural check run was published")
	}
	return run
}

func buildAddingImport(t *testing.T, path, baseSource, headSource string) application.BuildDocument {
	t.Helper()

	locator := gatedLocator(t)
	build := buildOverRecorded(t, locator, domain.File{
		Path: path, Action: domain.FileActionModified, Additions: 2, PatchBytes: 64,
	})
	build.Contents = recordedAtBothEnds(t, locator, path, baseSource, headSource)
	build.Extractor = golang.Extractor{}
	return build
}
