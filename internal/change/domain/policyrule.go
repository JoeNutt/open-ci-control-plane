// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

var ErrUnknownPolicyRule = errors.New("catalog policy names no rule this control plane ships")

type PolicyRule int

const DirectionRule PolicyRule = 1
