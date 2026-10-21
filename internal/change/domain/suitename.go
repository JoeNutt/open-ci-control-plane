// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

const MaxSuiteNameBytes = 512

var ErrMalformedSuiteName = errors.New("suite is not a Go import path")

type SuiteName struct {
	name string
}

func ParseSuiteName(name string) (SuiteName, error) {
	if len(name) > MaxSuiteNameBytes {
		return SuiteName{}, ErrMalformedSuiteName
	}
	return SuiteName{name: name}, nil
}
