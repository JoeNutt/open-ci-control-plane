// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

const MaxGoImportPathBytes = 512

var ErrMalformedGoImportPath = errors.New("import path is not a Go import path")

type GoImportPath struct {
	value string
}

func ParseGoImportPath(stated string) (GoImportPath, error) {
	if len(stated) > MaxGoImportPathBytes || !isSlashSeparatedNames(stated) {
		return GoImportPath{}, fmt.Errorf("%w: %d bytes", ErrMalformedGoImportPath, len(stated))
	}
	return GoImportPath{value: stated}, nil
}

func (path GoImportPath) String() string { return path.value }
