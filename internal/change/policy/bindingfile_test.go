// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"errors"
	"maps"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

func TestParseBindingFileReadsTheLiveBinding(t *testing.T) {
	parsed, err := policy.ParseBindingFile(fixture(t, liveBindingPath))
	if err != nil {
		t.Fatalf("parse binding file: %v", err)
	}
	if parsed.Repository().String() != liveRepository {
		t.Errorf("repository is %q", parsed.Repository())
	}
	if parsed.Workflow().String() != liveWorkflow {
		t.Errorf("workflow is %q", parsed.Workflow())
	}
	if len(parsed.AlwaysRun()) != 1 {
		t.Errorf("always-run suites are %v, want one", parsed.AlwaysRun())
	}
	policies := parsed.Policies()
	if len(policies) != 1 || policies[0].Use().String() != "clean-architecture" {
		t.Fatalf("policies are %v, want clean-architecture", policies)
	}
	if len(policies[0].ImportRoots()) != 1 || len(policies[0].Groups()) != 4 {
		t.Errorf("want one import root and four groups, got %v and %v",
			policies[0].ImportRoots(), policies[0].Groups())
	}
}

func TestParseBindingFileRefusesAFileItCannotTrust(t *testing.T) {
	refused := bindingsRefusedByTheirShape(t)
	maps.Copy(refused, map[string]refusedFile{
		"a YAML alias":                 {written: liveBindingWithAlias(t), reason: policy.ErrAliasOrAnchor},
		"one path bound to two groups": {written: liveBindingPlacingOnePathInTwoGroups(t), reason: domain.ErrPathInTwoGroups},
	})
	for form, refusal := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := policy.ParseBindingFile([]byte(refusal.written)); !errors.Is(err, refusal.reason) {
				t.Fatalf("%s gave %v, want %v", form, err, refusal.reason)
			}
		})
	}
}

func bindingsRefusedByTheirShape(t *testing.T) map[string]refusedFile {
	t.Helper()

	return map[string]refusedFile{
		"an import root with no path": {
			written: liveBindingWith(t, liveRootPathLine+"\n", ""),
			reason:  policy.ErrMissingKey,
		},
		"a later schema version": {
			written: liveBindingWith(t, liveSchemaVersionLine, laterSchemaVersionLine),
			reason:  policy.ErrUnsupportedSchemaVersion,
		},
		"no schema version": {
			written: liveBindingWith(t, liveSchemaVersionLine+"\n", ""),
			reason:  policy.ErrUnsupportedSchemaVersion,
		},
		"an unknown key": {written: liveBindingWithUnknownKey(t, "timeout"), reason: policy.ErrMalformedPolicyFile},
		"no workflow":    {written: liveBindingWithoutWorkflow(t), reason: domain.ErrMissingWorkflow},
	}
}

func liveBindingPlacingOnePathInTwoGroups(t *testing.T) string {
	t.Helper()

	return liveBindingWith(t, liveDriverPaths, `driver: ["cmd/**", "internal/change/rules/**"]`)
}

func liveBindingWithAlias(t *testing.T) string {
	t.Helper()

	return liveBindingWith(t, liveDriverPaths, `driver: &driver ["cmd/**"]`+"\n      extra: *driver")
}

func liveBindingWithoutWorkflow(t *testing.T) string {
	t.Helper()

	return liveBindingWith(t, liveWorkflowLine+"\n", "")
}

func liveBindingWithUnknownKey(t *testing.T, key string) string {
	t.Helper()

	return liveBindingWith(t, liveWorkflowLine, liveWorkflowLine+"\n  "+key+": 5")
}

func liveBindingWith(t *testing.T, written, replacement string) string {
	t.Helper()

	return fixtureWith(t, liveBindingPath, written, replacement)
}

func TestParseBindingFileReadsEachTestModeFromTheTestsBlock(t *testing.T) {
	for name, stated := range map[string]struct {
		written      string
		selection    domain.TestMode
		minimisation domain.TestMode
	}{
		"both stated": {
			written:   liveBindingWith(t, liveWorkflowLine, liveWorkflowLine+"\n  selection: enforce\n  minimisation: shadow"),
			selection: domain.TestModeEnforce, minimisation: domain.TestModeShadow,
		},
		"neither stated": {written: string(fixture(t, liveBindingPath)), selection: domain.TestModeShadow, minimisation: domain.TestModeOff},
	} {
		parsed, err := policy.ParseBindingFile([]byte(stated.written))

		if err != nil || parsed.SelectionMode() != stated.selection || parsed.MinimisationMode() != stated.minimisation {
			t.Errorf("%s: modes %q and %q (%v), want %q and %q", name,
				parsed.SelectionMode(), parsed.MinimisationMode(), err, stated.selection, stated.minimisation)
		}
	}
}
