// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const DecisionIDEntropyBytes = 16

const (
	decisionIDPrefix    = "dec_"
	decisionIDHexDigits = 2 * DecisionIDEntropyBytes
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

func DecisionIDFrom(entropy [DecisionIDEntropyBytes]byte) DecisionID {
	return DecisionID{value: decisionIDPrefix + hex.EncodeToString(entropy[:])}
}

func (id DecisionID) String() string { return id.value }
