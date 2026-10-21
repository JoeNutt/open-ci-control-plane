// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "strings"

const (
	pathSegmentCharacters = asciiAlphanumerics + "-._~"
	pathSegmentDot        = "."
)

var pathSegmentRule = segmentRule{
	maxBytes:          DeclaredLimits().MaxPathBytes,
	firstCharacters:   pathSegmentCharacters,
	wholeCharacters:   pathSegmentCharacters,
	requiresOneBeyond: pathSegmentDot,
}

func isSlashSeparatedNames(stated string) bool {
	for segment := range strings.SplitSeq(stated, pathSeparator) {
		if !pathSegmentRule.admits(segment) {
			return false
		}
	}
	return true
}
