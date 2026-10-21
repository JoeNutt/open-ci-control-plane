// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"errors"
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
	refused := map[string]struct {
		written string
		reason  error
	}{
		"an import root with no path": {
			written: liveBindingWith(t, "        path: \"\"\n", ""),
			reason:  policy.ErrMissingKey,
		},
		"a later schema version": {
			written: liveBindingWith(t, "schema_version: 1", "schema_version: 2"),
			reason:  policy.ErrUnsupportedSchemaVersion,
		},
		"no schema version": {
			written: liveBindingWith(t, "schema_version: 1\n", ""),
			reason:  policy.ErrUnsupportedSchemaVersion,
		},
		"an unknown key": {
			written: liveBindingWithUnknownKey(t, "timeout"),
			reason:  policy.ErrMalformedPolicyFile,
		},
		"a YAML alias": {
			written: liveBindingWithAlias(t),
			reason:  policy.ErrAliasOrAnchor,
		},
		"no workflow": {
			written: liveBindingWithoutWorkflow(t),
			reason:  domain.ErrMissingWorkflow,
		},
		"one path bound to two groups": {
			written: liveBindingPlacingOnePathInTwoGroups(t),
			reason:  domain.ErrPathInTwoGroups,
		},
	}
	for form, refusal := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := policy.ParseBindingFile([]byte(refusal.written)); !errors.Is(err, refusal.reason) {
				t.Fatalf("%s gave %v, want %v", form, err, refusal.reason)
			}
		})
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
