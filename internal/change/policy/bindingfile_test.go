// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

const liveBindingPath = "testdata/bindings/JoeNutt/open-ci-control-plane.yaml"

func TestParseBindingFileReadsTheLiveBinding(t *testing.T) {
	parsed, err := policy.ParseBindingFile(fixture(t, liveBindingPath))
	if err != nil {
		t.Fatalf("parse binding file: %v", err)
	}
	if parsed.Repository().String() != "JoeNutt/open-ci-control-plane" {
		t.Errorf("repository is %q", parsed.Repository())
	}
	if parsed.Workflow().String() != "open-ci-tests.yml" {
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
