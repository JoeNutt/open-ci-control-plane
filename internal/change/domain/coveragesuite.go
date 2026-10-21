// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
)

var ErrMalformedCoverageSuite = errors.New("suite is not a C-12 coverage suite")

type CoverageSuite struct {
	importPath GoImportPath
}

func ParseCoverageSuite(stated string) (CoverageSuite, error) {
	importPath, err := ParseGoImportPath(stated)
	if err != nil {
		return CoverageSuite{}, fmt.Errorf("%w: %w", ErrMalformedCoverageSuite, err)
	}
	if strings.Trim(stated, asciiAlphanumerics+"._/-") != "" {
		return CoverageSuite{}, ErrMalformedCoverageSuite
	}
	return CoverageSuite{importPath: importPath}, nil
}
