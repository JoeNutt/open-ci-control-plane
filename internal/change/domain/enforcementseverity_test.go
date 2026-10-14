// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestEveryEnforcementSeverityIsNamed(t *testing.T) {
	declared := map[string]domain.EnforcementSeverity{
		"deny":    domain.EnforcementSeverityDeny,
		"warn":    domain.EnforcementSeverityWarn,
		"dry_run": domain.EnforcementSeverityDryRun,
	}

	for name, severity := range declared {
		t.Run(name, func(t *testing.T) {
			parsed, err := domain.ParseEnforcementSeverity(name)
			if err != nil {
				t.Fatalf("parse %q: %v", name, err)
			}
			if parsed != severity {
				t.Errorf("%q parsed to %q, want %q", name, parsed, severity)
			}
		})
	}
}

func TestParseEnforcementSeverityRefusesAnythingElse(t *testing.T) {
	for _, name := range []string{"", "high", "critical", "dry-run", "dryrun", "DENY", "block"} {
		t.Run(name, func(t *testing.T) {
			_, err := domain.ParseEnforcementSeverity(name)
			if !errors.Is(err, domain.ErrUnknownEnforcementSeverity) {
				t.Fatalf("%q gave %v, want ErrUnknownEnforcementSeverity", name, err)
			}
		})
	}
}
