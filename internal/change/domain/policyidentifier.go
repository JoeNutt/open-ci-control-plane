// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

const MaxPolicyIdentifierBytes = 64

var policyIdentifierRule = segmentRule{
	maxBytes:        MaxPolicyIdentifierBytes,
	firstCharacters: asciiAlphanumerics,
	wholeCharacters: asciiAlphanumerics + "-_",
}

func admittedPolicyIdentifier(stated string, malformed error) (string, error) {
	if !policyIdentifierRule.admits(stated) {
		return "", fmt.Errorf("%w: %d bytes", malformed, len(stated))
	}
	return stated, nil
}
