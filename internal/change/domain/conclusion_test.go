// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestEveryConclusionACheckRunMayCarryIsNamed(t *testing.T) {
	declared := map[string]domain.Conclusion{
		"success": domain.ConclusionSuccess,
		"failure": domain.ConclusionFailure,
		"neutral": domain.ConclusionNeutral,
	}

	for name, conclusion := range declared {
		t.Run(name, func(t *testing.T) {
			parsed, err := domain.ParseConclusion(name)
			if err != nil {
				t.Fatalf("parse %q: %v", name, err)
			}
			if parsed != conclusion {
				t.Errorf("%q parsed to %q, want %q", name, parsed, conclusion)
			}
		})
	}
}

func TestParseConclusionRefusesAnythingElse(t *testing.T) {
	for _, name := range []string{"", "skipped", "cancelled", "action_required", "stale", "SUCCESS"} {
		t.Run(name, func(t *testing.T) {
			if _, err := domain.ParseConclusion(name); !errors.Is(err, domain.ErrUnknownConclusion) {
				t.Fatalf("%q gave %v, want ErrUnknownConclusion", name, err)
			}
		})
	}
}
