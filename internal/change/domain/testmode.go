// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

type TestMode string

const (
	TestModeOff     TestMode = "off"
	TestModeShadow  TestMode = "shadow"
	TestModeEnforce TestMode = "enforce"
)

var testModes = []TestMode{TestModeOff, TestModeShadow, TestModeEnforce}

var ErrUnknownTestMode = errors.New("the mode is not off, shadow or enforce")

func ParseTestMode(stated string) (TestMode, error) {
	if !slices.Contains(testModes, TestMode(stated)) {
		return "", fmt.Errorf("%w: %q", ErrUnknownTestMode, stated)
	}
	return TestMode(stated), nil
}

func (mode TestMode) String() string { return string(mode) }
