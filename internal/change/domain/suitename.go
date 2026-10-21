// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

const MaxSuiteNameBytes = 512

var ErrMalformedSuiteName = errors.New("suite is not a Go import path")

type SuiteName struct {
	name string
}

func ParseSuiteName(name string) (SuiteName, error) {
	if len(name) > MaxSuiteNameBytes {
		return SuiteName{}, fmt.Errorf("%w: %d bytes exceeds %d",
			ErrMalformedSuiteName, len(name), MaxSuiteNameBytes)
	}
	if !fs.ValidPath(name) || name == "." || strings.Trim(name, asciiAlphanumerics+"._/-") != "" {
		return SuiteName{}, ErrMalformedSuiteName
	}
	return SuiteName{name: name}, nil
}
