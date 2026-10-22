// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"encoding/base64"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	contentsLineCharacters     = 60
	escapedLineBreakBytes      = 2
	contentsPathFields         = 7
	percentEncodedByteWidth    = 3
	contentsFixedEnvelopeBytes = 4 << 10
)

func MaxResponseBytes() int {
	limits := domain.DeclaredLimits()
	encoded := base64.StdEncoding.EncodedLen(limits.MaxSourceBytes)
	lineBreaks := linesOf(encoded) * escapedLineBreakBytes
	envelope := contentsPathFields*percentEncodedByteWidth*limits.MaxPathBytes + contentsFixedEnvelopeBytes
	return encoded + lineBreaks + envelope
}

func linesOf(characters int) int {
	return (characters + contentsLineCharacters - 1) / contentsLineCharacters
}
