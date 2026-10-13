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
