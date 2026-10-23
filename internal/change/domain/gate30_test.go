// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var gatedAt = time.Date(2026, 11, 30, 12, 0, 0, 0, time.UTC)

func TestAModeWithAnEscapeInTheLast30DaysIsNotEnforcedAndNamesEscapeRecorded(t *testing.T) {
	escapes := []domain.Escape{{Mode: domain.ShadowedSelection, At: gatedAt.Add(-29 * 24 * time.Hour)}}

	verdict := domain.EnforcementOf(domain.ShadowedSelection, nil, escapes, gatedAt)

	if verdict.IsEnforced || verdict.Trigger != domain.PlanTriggerEscapeRecorded {
		t.Errorf("a mode with an escape 29 days ago gave %+v, want not enforced with %q", verdict, domain.PlanTriggerEscapeRecorded)
	}
}
