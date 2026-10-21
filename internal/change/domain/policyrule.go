// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"slices"
)

var ErrUnknownPolicyRule = errors.New("catalog policy names no rule this control plane ships")

type PolicyRule int

const (
	DirectionRule PolicyRule = iota + 1
)

var shippedPolicyRules = []PolicyRule{DirectionRule}

func (rule PolicyRule) isShipped() bool { return slices.Contains(shippedPolicyRules, rule) }
