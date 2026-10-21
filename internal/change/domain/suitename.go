// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
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
	return SuiteName{name: name}, nil
}
