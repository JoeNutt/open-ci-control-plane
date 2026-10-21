// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"os"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

func TestParseCatalogFileReadsTheLiveCatalog(t *testing.T) {
	written, err := os.ReadFile("testdata/catalog/clean-architecture.yaml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	parsed, err := policy.ParseCatalogFile(written)
	if err != nil {
		t.Fatalf("parse catalog file: %v", err)
	}
	if parsed.Name().String() != "clean-architecture" {
		t.Errorf("name is %q, want clean-architecture", parsed.Name())
	}
	if parsed.Direction() != domain.LowerOrderMayNotImportHigher {
		t.Errorf("direction is %v, want lower order may not import higher", parsed.Direction())
	}
	groups := parsed.Groups()
	if len(groups) != 4 || groups[3].Name().String() != "driver" || groups[3].Order() != 3 {
		t.Errorf("groups are %v, want four ending in driver at 3", groups)
	}
}
