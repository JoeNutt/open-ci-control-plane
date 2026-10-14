// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

type Effect string

const (
	EffectBlock  Effect = "block"
	EffectAllow  Effect = "allow"
	EffectReview Effect = "review"
)

var declaredEffects = []Effect{EffectBlock, EffectAllow, EffectReview}

var ErrUnknownEffect = errors.New("unknown decision effect")

func ParseEffect(name string) (Effect, error) {
	return parseDeclaration(name, declaredEffects, ErrUnknownEffect)
}
