// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
)

const MaxGoImportPathBytes = 512

var ErrMalformedGoImportPath = errors.New("import path is not a Go import path")

type GoImportPath struct {
	value string
}

var importElementRule = segmentRule{
	maxBytes:          DeclaredLimits().MaxPathBytes,
	firstCharacters:   pathSegmentCharacters + "+",
	wholeCharacters:   pathSegmentCharacters + "+",
	requiresOneBeyond: pathSegmentDot,
}

func ParseGoImportPath(stated string) (GoImportPath, error) {
	if len(stated) > MaxGoImportPathBytes {
		return GoImportPath{}, fmt.Errorf("%w: %d bytes", ErrMalformedGoImportPath, len(stated))
	}
	for segment := range strings.SplitSeq(stated, pathSeparator) {
		if !importElementRule.admits(segment) {
			return GoImportPath{}, fmt.Errorf("%w: %d bytes", ErrMalformedGoImportPath, len(stated))
		}
	}
	return GoImportPath{value: stated}, nil
}

func (path GoImportPath) String() string { return path.value }
