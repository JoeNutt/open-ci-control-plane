// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseFileActionAcceptsEveryDeclaredAction(t *testing.T) {
	declared := map[string]domain.FileAction{
		"added":    domain.FileActionAdded,
		"modified": domain.FileActionModified,
		"removed":  domain.FileActionRemoved,
		"renamed":  domain.FileActionRenamed,
		"copied":   domain.FileActionCopied,
		"changed":  domain.FileActionChanged,
	}

	for name, want := range declared {
		t.Run(name, func(t *testing.T) {
			action, err := domain.ParseFileAction(name)
			if err != nil {
				t.Fatalf("parse %q: %v", name, err)
			}
			if action != want {
				t.Fatalf("parsed %q as %q, want %q", name, action, want)
			}
		})
	}
}

func TestParseFileActionRefusesAnUnmappedValue(t *testing.T) {
	action, err := domain.ParseFileAction("exfiltrated")

	if !errors.Is(err, domain.ErrUnknownFileAction) {
		t.Fatalf("want ErrUnknownFileAction, got %v", err)
	}
	if action != "" {
		t.Fatalf("want no action alongside the refusal, got %q", action)
	}
}
