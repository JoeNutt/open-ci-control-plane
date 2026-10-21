// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
)

const (
	MaxSuiteNameBytes   = 512
	suiteNameCharacters = asciiAlphanumerics + "._/-"
)

var ErrMalformedSuiteName = errors.New("suite is not a Go import path")

var (
	errNonCanonicalSuiteName = fmt.Errorf("%w: it is not one canonical slash-separated path", ErrMalformedSuiteName)
	errSuiteNameCharacter    = fmt.Errorf("%w: it holds a character outside [A-Za-z0-9._/-]", ErrMalformedSuiteName)
)

type SuiteName struct {
	name string
}

func ParseSuiteName(name string) (SuiteName, error) {
	if len(name) > MaxSuiteNameBytes {
		return SuiteName{}, fmt.Errorf("%w: %d bytes exceeds %d",
			ErrMalformedSuiteName, len(name), MaxSuiteNameBytes)
	}
	if !isCanonicalSlashPath(name) {
		return SuiteName{}, errNonCanonicalSuiteName
	}
	if strings.Trim(name, suiteNameCharacters) != "" {
		return SuiteName{}, errSuiteNameCharacter
	}
	return SuiteName{name: name}, nil
}
