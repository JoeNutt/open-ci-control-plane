// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

const MaxPolicyNameBytes = 64

var ErrMalformedPolicyName = errors.New("policy name is not an identifier")

var policyIdentifierRule = segmentRule{
	maxBytes:        MaxPolicyNameBytes,
	firstCharacters: asciiAlphanumerics,
	wholeCharacters: asciiAlphanumerics + "-_",
}

type PolicyName struct {
	value string
}

func ParsePolicyName(stated string) (PolicyName, error) {
	if !policyIdentifierRule.admits(stated) {
		return PolicyName{}, fmt.Errorf("%w: %d bytes", ErrMalformedPolicyName, len(stated))
	}
	return PolicyName{value: stated}, nil
}

func (name PolicyName) String() string { return name.value }
