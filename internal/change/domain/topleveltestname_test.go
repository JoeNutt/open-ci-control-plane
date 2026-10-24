// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseTopLevelTestNameFollowsTheGoToolsRuleForATestName(t *testing.T) {
	for name, isTest := range map[string]bool{
		"Test":         true,
		"TestÜber":     true,
		"TestAdmits":   true,
		"Test_admits":  true,
		"Test2Admits":  true,
		"Testify":      false,
		"Testé":        false,
		"test":         false,
		"BenchmarkX":   false,
		"Test-Admits":  false,
		"TestAdmits/x": false,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := domain.ParseTopLevelTestName(name)

			if (err == nil) != isTest {
				t.Fatalf("%q parsed with %v, want a test name %v", name, err, isTest)
			}
		})
	}
}
