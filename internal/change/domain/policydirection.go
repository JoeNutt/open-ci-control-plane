// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"slices"
)

var ErrUnknownPolicyDirection = errors.New("catalog policy names no direction this control plane ships")

type PolicyDirection int

const (
	LowerOrderMayNotImportHigher PolicyDirection = iota + 1
)

var shippedPolicyDirections = []PolicyDirection{LowerOrderMayNotImportHigher}

func (direction PolicyDirection) isShipped() bool {
	return slices.Contains(shippedPolicyDirections, direction)
}
