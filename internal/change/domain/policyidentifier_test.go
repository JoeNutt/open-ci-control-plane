// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func malformedIdentifiers() map[string]string {
	return map[string]string{
		"empty":               "",
		"a space":             "clean architecture",
		"a leading hyphen":    "-clean",
		"a path separator":    "clean/architecture",
		"a non-ASCII letter":  "cléan",
		"one byte over bound": strings.Repeat("a", domain.MaxPolicyIdentifierBytes+1),
	}
}
