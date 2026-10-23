// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestATestModeParsesFromOffShadowOrEnforceOnly(t *testing.T) {
	for stated, isAccepted := range map[string]bool{
		"off": true, "shadow": true, "enforce": true,
		"": false, "Shadow": false, "enforced": false, "on": false,
	} {
		parsed, err := domain.ParseTestMode(stated)

		if isAccepted && (err != nil || parsed.String() != stated) {
			t.Errorf("mode %q parsed to %q (%v), want it accepted as itself", stated, parsed, err)
		}
		if !isAccepted && err == nil {
			t.Errorf("mode %q was accepted as %q, want it refused", stated, parsed)
		}
	}
}
