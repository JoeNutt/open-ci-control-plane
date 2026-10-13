// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestZeroDocumentEmitsEmptyPathsAsList(t *testing.T) {
	encoded, err := json.Marshal(domain.Document{})
	if err != nil {
		t.Fatalf("marshal zero document: %v", err)
	}
	if !strings.Contains(string(encoded), `"paths":[]`) {
		t.Fatalf(`want "paths":[] in %s`, encoded)
	}
}

func TestPathsEmitLexicographicallySorted(t *testing.T) {
	document := domain.Document{Diff: domain.Diff{Paths: domain.Paths{
		"internal/http/routes.go",
		"infra/iam.tf",
		"internal/auth/middleware.go",
	}}}

	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal document: %v", err)
	}

	want := `"paths":["infra/iam.tf","internal/auth/middleware.go","internal/http/routes.go"]`
	if !strings.Contains(string(encoded), want) {
		t.Fatalf("want %s in %s", want, encoded)
	}
}
