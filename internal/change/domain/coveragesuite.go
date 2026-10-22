// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
)

const coverageSuiteCharacters = asciiAlphanumerics + "._/-"

var ErrMalformedCoverageSuite = errors.New("suite is not a C-12 coverage suite")

var errCoverageSuiteCharacter = fmt.Errorf("%w: it holds a character outside [A-Za-z0-9._/-]", ErrMalformedCoverageSuite)

type CoverageSuite struct {
	importPath GoImportPath
}

func ParseCoverageSuite(stated string) (CoverageSuite, error) {
	importPath, err := ParseGoImportPath(stated)
	if err != nil {
		return CoverageSuite{}, fmt.Errorf("%w: %w", ErrMalformedCoverageSuite, err)
	}
	if strings.Trim(stated, coverageSuiteCharacters) != "" {
		return CoverageSuite{}, errCoverageSuiteCharacter
	}
	return CoverageSuite{importPath: importPath}, nil
}

func (suite CoverageSuite) String() string { return suite.importPath.String() }
