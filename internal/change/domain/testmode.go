// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

type TestMode struct {
	value string
}

func ParseTestMode(stated string) (TestMode, error) {
	if stated != "off" && stated != "shadow" && stated != "enforce" {
		return TestMode{}, errors.New("not a test mode")
	}
	return TestMode{value: stated}, nil
}

func (mode TestMode) String() string { return mode.value }
