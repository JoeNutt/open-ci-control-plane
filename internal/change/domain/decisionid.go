// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
)

const (
	decisionIDPrefix    = "dec_"
	decisionIDHexDigits = 32
)

var ErrMalformedDecisionID = errors.New("decision identifier is malformed")

type DecisionID struct {
	value string
}

func ParseDecisionID(stated string) (DecisionID, error) {
	if len(stated) != len(decisionIDPrefix)+decisionIDHexDigits {
		return DecisionID{}, fmt.Errorf("%w: %d bytes", ErrMalformedDecisionID, len(stated))
	}
	digits, prefixed := strings.CutPrefix(stated, decisionIDPrefix)
	if !prefixed || !isLowerHex(digits) {
		return DecisionID{}, fmt.Errorf("%w: %q", ErrMalformedDecisionID, stated)
	}
	return DecisionID{value: stated}, nil
}

func (id DecisionID) String() string { return id.value }

func isLowerHex(digits string) bool {
	for index := 0; index < len(digits); index++ {
		if !isLowerHexDigit(digits[index]) {
			return false
		}
	}
	return true
}

func isLowerHexDigit(character byte) bool {
	return ('0' <= character && character <= '9') || ('a' <= character && character <= 'f')
}
