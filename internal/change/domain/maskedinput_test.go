// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestMaskedInputOfOmitsAFieldTheMaskNames(t *testing.T) {
	document := domain.Document{}
	document.PR.Author.Login = "octocat"
	document.Diff.Sensitive = domain.Declared[string]{"/pr/author/login"}

	masked, err := domain.MaskedInputOf(document)
	if err != nil {
		t.Fatalf("mask: %v", err)
	}

	encoded, err := json.Marshal(masked)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if strings.Contains(string(encoded), "octocat") {
		t.Errorf("masked input still carries the sensitive value: %s", encoded)
	}
}
