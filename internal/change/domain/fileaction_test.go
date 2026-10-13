// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseFileActionRefusesAnUnmappedValue(t *testing.T) {
	action, err := domain.ParseFileAction("exfiltrated")

	if !errors.Is(err, domain.ErrUnknownFileAction) {
		t.Fatalf("want ErrUnknownFileAction, got %v", err)
	}
	if action != "" {
		t.Fatalf("want no action alongside the refusal, got %q", action)
	}
}
