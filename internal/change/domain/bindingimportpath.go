// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
)

const MaxGoImportPathBytes = 512

var ErrMalformedGoImportPath = errors.New("import path is not a Go import path")

var importPathSegmentRule = segmentRule{
	maxBytes:          MaxGoImportPathBytes,
	firstCharacters:   asciiAlphanumerics + "-._~",
	wholeCharacters:   asciiAlphanumerics + "-._~",
	requiresOneBeyond: ".",
}

type GoImportPath struct {
	value string
}

func ParseGoImportPath(stated string) (GoImportPath, error) {
	if len(stated) > MaxGoImportPathBytes {
		return GoImportPath{}, fmt.Errorf("%w: %d bytes exceeds %d", ErrMalformedGoImportPath, len(stated), MaxGoImportPathBytes)
	}
	for segment := range strings.SplitSeq(stated, pathSeparator) {
		if !importPathSegmentRule.admits(segment) {
			return GoImportPath{}, fmt.Errorf("%w: %d bytes", ErrMalformedGoImportPath, len(stated))
		}
	}
	return GoImportPath{value: stated}, nil
}

func (path GoImportPath) String() string { return path.value }
