// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParsePolicyNameKeepsTheName(t *testing.T) {
	name, err := domain.ParsePolicyName("clean-architecture")
	if err != nil {
		t.Fatalf("parse policy name: %v", err)
	}
	if name.String() != "clean-architecture" {
		t.Errorf("name is %q, want clean-architecture", name)
	}
}
